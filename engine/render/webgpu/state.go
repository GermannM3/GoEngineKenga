//go:build webgpu && !js

package webgpu

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"
	"os"

	"github.com/cogentcore/webgpu/wgpu"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/render"
)

//go:embed shader.wgsl
var shaderWGSL string

//go:embed shader_skinned.wgsl
var shaderSkinnedWGSL string

//go:embed point_shadow.wgsl
var pointShadowWGSL string

//go:embed point_shadow_skinned.wgsl
var pointShadowSkinnedWGSL string

//go:embed postprocess.wgsl
var postprocessWGSL string

type state struct {
	instance *wgpu.Instance
	adapter  *wgpu.Adapter
	surface  *wgpu.Surface
	device   *wgpu.Device
	queue    *wgpu.Queue
	config   *wgpu.SurfaceConfiguration
	pipeline *wgpu.RenderPipeline
	scene    *sceneState
}

func initState[T interface{ GetSize() (int, int) }](window T, sd *wgpu.SurfaceDescriptor) (s *state, err error) {
	defer func() {
		if err != nil && s != nil {
			s.Destroy()
			s = nil
		}
	}()
	s = &state{}

	s.instance = wgpu.CreateInstance(nil)
	s.surface = s.instance.CreateSurface(sd)

	s.adapter, err = s.instance.RequestAdapter(&wgpu.RequestAdapterOptions{
		CompatibleSurface: s.surface,
	})
	if err != nil {
		return s, err
	}
	defer s.adapter.Release()

	s.device, err = s.adapter.RequestDevice(nil)
	if err != nil {
		return s, err
	}
	s.queue = s.device.GetQueue()

	caps := s.surface.GetCapabilities(s.adapter)
	width, height := window.GetSize()
	s.config = &wgpu.SurfaceConfiguration{
		Usage:       wgpu.TextureUsageRenderAttachment,
		Format:      caps.Formats[0],
		Width:       uint32(width),
		Height:      uint32(height),
		PresentMode: wgpu.PresentModeFifo,
		AlphaMode:   caps.AlphaModes[0],
	}
	s.surface.Configure(s.adapter, s.device, s.config)

	if err := s.initSceneState(); err != nil {
		return s, err
	}
	return s, nil
}

func (s *state) Resize(width, height int) {
	if width > 0 && height > 0 {
		s.config.Width = uint32(width)
		s.config.Height = uint32(height)
		s.surface.Configure(s.adapter, s.device, s.config)
	}
}

func (s *state) RenderScene(frame *render.Frame, resolver *asset.Resolver) error {
	if s.scene == nil {
		return nil
	}
	if frame == nil || frame.World == nil {
		nextTexture, err := s.surface.GetCurrentTexture()
		if err != nil {
			return err
		}
		view, err := nextTexture.CreateView(nil)
		if err != nil {
			return err
		}
		defer view.Release()
		encoder, err := s.device.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{})
		if err != nil {
			return err
		}
		defer encoder.Release()
		rp := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
			ColorAttachments: []wgpu.RenderPassColorAttachment{
				{View: view, LoadOp: wgpu.LoadOpClear, StoreOp: wgpu.StoreOpStore, ClearValue: wgpu.Color{R: 0.06, G: 0.07, B: 0.09, A: 1.0}},
			},
		})
		rp.End()
		rp.Release()
		cb, err := encoder.Finish(nil)
		if err != nil {
			return err
		}
		defer cb.Release()
		s.queue.Submit(cb)
		s.surface.Present()
		return nil
	}
	// Инвалидация mesh cache при hot-reload ассетов
	if frame.InvalidateMeshCache {
		frame.InvalidateMeshCache = false
		if s.scene != nil {
			for _, c := range s.scene.meshCache {
				if c != nil && c.vertexBuf != nil {
					c.vertexBuf.Release()
				}
			}
			s.scene.meshCache = make(map[string]*cachedMesh)
			for _, c := range s.scene.meshSkinnedCache {
				if c != nil && c.vertexBuf != nil {
					c.vertexBuf.Release()
				}
			}
			s.scene.meshSkinnedCache = make(map[string]*cachedMesh)
		}
	}

	width, height := int(s.config.Width), int(s.config.Height)
	nextTexture, err := s.surface.GetCurrentTexture()
	if err != nil {
		return err
	}
	view, err := nextTexture.CreateView(nil)
	if err != nil {
		return err
	}
	defer view.Release()

	cc := wgpu.Color{R: 0.06, G: 0.07, B: 0.09, A: 1.0}
	if frame != nil && frame.ClearColor.A > 0 {
		cc = wgpu.Color{
			R: float64(frame.ClearColor.R) / 255,
			G: float64(frame.ClearColor.G) / 255,
			B: float64(frame.ClearColor.B) / 255,
			A: 1.0,
		}
	}

	encoder, err := s.device.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "encoder"})
	if err != nil {
		return err
	}
	defer encoder.Release()

	// Буферы инстансов живут до Submit (Release до Submit = use-after-destroy в wgpu).
	var transientBuffers []*wgpu.Buffer

	sc := s.scene
	pbrData, ok := getPBRSceneData(frame.World, width, height)
	if !ok {
		cam := render.NewCamera3D()
		cam.SetPosition(emath.Vec3{X: 0, Y: 0, Z: 5})
		cam.SetTarget(emath.Vec3{X: 0, Y: 0, Z: 0})
		cam.SetAspectRatio(float32(width) / float32(height))
		pbrData.viewProj = cam.GetViewProjectionMatrix()
		pbrData.camPos = []float32{0, 0, 5}
		pbrData.lightDir = []float32{0.5, 1.0, 0.3}
		pbrData.lightColor = []float32{1.0, 1.0, 1.0}
		pbrData.lightIntensity = 1.0
		pbrData.ambient = 0.15
	}

	lightViewProj := buildLightViewProj(pbrData.lightDir)
	ubBytes := make([]byte, pbrUniformsSize)
	shadowUbBytes := make([]byte, 128)
	boneBytes := make([]byte, boneUniformsSize)
	frustum := render.ExtractFrustum(pbrData.viewProj)

	// Shadow pass
	if sc.shadowView != nil && sc.shadowPipeline != nil {
		shadowPass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
			ColorAttachments: []wgpu.RenderPassColorAttachment{},
			DepthStencilAttachment: &wgpu.RenderPassDepthStencilAttachment{
				View:            sc.shadowView,
				DepthLoadOp:     wgpu.LoadOpClear,
				DepthStoreOp:    wgpu.StoreOpStore,
				DepthClearValue: 1.0,
			},
		})
		shadowPass.SetPipeline(sc.shadowPipeline)
		copy(shadowUbBytes[0:64], matrixToBytes(lightViewProj))
		shadowPass.SetBindGroup(0, sc.shadowBindGroup, nil)
		skinnedShadowSet := false

		for _, id := range frame.World.Entities() {
			mr, hasMR := frame.World.GetMeshRenderer(id)
			if !hasMR || mr.MeshAssetID == "" {
				continue
			}
			tr, hasTr := frame.World.GetTransform(id)
			if !hasTr {
				tr = ecs.Transform{Scale: emath.Vec3{X: 1, Y: 1, Z: 1}}
			}
			sx, sy, sz := tr.Scale.X, tr.Scale.Y, tr.Scale.Z
			if sx < 0.01 {
				sx = 1
			}
			if sy < 0.01 {
				sy = 1
			}
			if sz < 0.01 {
				sz = 1
			}
			radius := float32(math.Sqrt(float64(sx*sx + sy*sy + sz*sz)))
			if !frustum.SphereInFrustum(tr.Position, radius) {
				continue
			}
			skinned := isMeshSkinned(resolver, mr.MeshAssetID)
			var vb *wgpu.Buffer
			var vc uint32
			if skinned {
				vb, vc = sc.getOrCreateSkinnedMeshBuffer(resolver, mr.MeshAssetID)
				if sc.shadowSkinnedPipeline != nil {
					shadowPass.SetPipeline(sc.shadowSkinnedPipeline)
					if !skinnedShadowSet {
						shadowPass.SetBindGroup(0, sc.shadowSkinnedBindGroup, nil)
						skinnedShadowSet = true
					}
					// Per-entity bone matrices (если анимируется — из Animator, иначе bind pose)
					if anim, ok := frame.World.GetAnimator(id); ok {
						writeBoneMatrices(boneBytes, anim.BoneMatrices)
					} else {
						writeBoneMatrices(boneBytes, nil)
					}
					s.queue.WriteBuffer(sc.boneUniform, 0, boneBytes)
				}
			} else {
				vb, vc = sc.getOrCreateMeshBuffer(resolver, mr.MeshAssetID)
				shadowPass.SetPipeline(sc.shadowPipeline)
				if skinnedShadowSet {
					shadowPass.SetBindGroup(0, sc.shadowBindGroup, nil)
					skinnedShadowSet = false
				}
			}
			if vb == nil {
				continue
			}
			model := buildModelMatrix(&tr)
			copy(shadowUbBytes[64:128], matrixToBytes(model))
			if skinned && sc.shadowSkinnedUniform != nil {
				s.queue.WriteBuffer(sc.shadowSkinnedUniform, 0, shadowUbBytes)
			} else {
				s.queue.WriteBuffer(sc.shadowUniform, 0, shadowUbBytes)
			}
			shadowPass.SetVertexBuffer(0, vb, 0, wgpu.WholeSize)
			shadowPass.Draw(vc, 1, 0, 0)
		}
		shadowPass.End()
		shadowPass.Release()
	}

	// Point light shadow pass: cubemap из depth-array (6 граней), линейная глубина.
	if sc.pointShadowPipeline != nil && len(pbrData.pointLightPos) == 3 && pbrData.pointLightIntensity > 0 {
		lightPos := emath.Vec3{X: pbrData.pointLightPos[0], Y: pbrData.pointLightPos[1], Z: pbrData.pointLightPos[2]}
		far := pbrData.pointLightRange
		if far <= 0 {
			far = 10
		}
		pointUbBytes := make([]byte, 144)
		binary.LittleEndian.PutUint32(pointUbBytes[128:], math.Float32bits(lightPos.X))
		binary.LittleEndian.PutUint32(pointUbBytes[132:], math.Float32bits(lightPos.Y))
		binary.LittleEndian.PutUint32(pointUbBytes[136:], math.Float32bits(lightPos.Z))
		binary.LittleEndian.PutUint32(pointUbBytes[140:], math.Float32bits(far))
		for face := 0; face < 6; face++ {
			faceVP := buildPointFaceViewProj(lightPos, face, far)
			facePass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
				ColorAttachments: []wgpu.RenderPassColorAttachment{},
				DepthStencilAttachment: &wgpu.RenderPassDepthStencilAttachment{
					View:            sc.pointShadowFaces[face],
					DepthLoadOp:     wgpu.LoadOpClear,
					DepthStoreOp:    wgpu.StoreOpStore,
					DepthClearValue: 1.0,
				},
			})
			facePass.SetPipeline(sc.pointShadowPipeline)
			facePass.SetBindGroup(0, sc.pointShadowBindGroup, nil)
			copy(pointUbBytes[0:64], matrixToBytes(faceVP))
			skinnedSet := false
			for _, id := range frame.World.Entities() {
				mr, hasMR := frame.World.GetMeshRenderer(id)
				if !hasMR || mr.MeshAssetID == "" {
					continue
				}
				tr, hasTr := frame.World.GetTransform(id)
				if !hasTr {
					tr = ecs.Transform{Scale: emath.Vec3{X: 1, Y: 1, Z: 1}}
				}
				skinned := isMeshSkinned(resolver, mr.MeshAssetID)
				var vb *wgpu.Buffer
				var vc uint32
				if skinned {
					vb, vc = sc.getOrCreateSkinnedMeshBuffer(resolver, mr.MeshAssetID)
					if sc.pointShadowSkinnedPipeline != nil {
						facePass.SetPipeline(sc.pointShadowSkinnedPipeline)
						if !skinnedSet {
							facePass.SetBindGroup(0, sc.pointShadowSkinnedBindGroup, nil)
							skinnedSet = true
						}
						if anim, ok := frame.World.GetAnimator(id); ok {
							writeBoneMatrices(boneBytes, anim.BoneMatrices)
						} else {
							writeBoneMatrices(boneBytes, nil)
						}
						s.queue.WriteBuffer(sc.boneUniform, 0, boneBytes)
					}
				} else {
					vb, vc = sc.getOrCreateMeshBuffer(resolver, mr.MeshAssetID)
					facePass.SetPipeline(sc.pointShadowPipeline)
					if skinnedSet {
						facePass.SetBindGroup(0, sc.pointShadowBindGroup, nil)
						skinnedSet = false
					}
				}
				if vb == nil {
					continue
				}
				model := buildModelMatrix(&tr)
				copy(pointUbBytes[64:128], matrixToBytes(model))
				if skinned && sc.pointShadowSkinnedUniform != nil {
					s.queue.WriteBuffer(sc.pointShadowSkinnedUniform, 0, pointUbBytes)
				} else {
					s.queue.WriteBuffer(sc.pointShadowUniform, 0, pointUbBytes)
				}
				facePass.SetVertexBuffer(0, vb, 0, wgpu.WholeSize)
				facePass.Draw(vc, 1, 0, 0)
			}
			facePass.End()
			facePass.Release()
		}
	}

	// Spot light shadow pass: перспективная карта из прожектора.
	if sc.spotShadowView != nil && len(pbrData.spotLightDir) == 3 && pbrData.spotLightIntensity > 0 {
		spotPass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
			ColorAttachments: []wgpu.RenderPassColorAttachment{},
			DepthStencilAttachment: &wgpu.RenderPassDepthStencilAttachment{
				View:            sc.spotShadowView,
				DepthLoadOp:     wgpu.LoadOpClear,
				DepthStoreOp:    wgpu.StoreOpStore,
				DepthClearValue: 1.0,
			},
		})
		spotPass.SetPipeline(sc.shadowPipeline)
		copy(shadowUbBytes[0:64], matrixToBytes(pbrData.spotLightViewProj))
		spotPass.SetBindGroup(0, sc.shadowBindGroup, nil)
		skinnedSet := false
		for _, id := range frame.World.Entities() {
			mr, hasMR := frame.World.GetMeshRenderer(id)
			if !hasMR || mr.MeshAssetID == "" {
				continue
			}
			tr, hasTr := frame.World.GetTransform(id)
			if !hasTr {
				tr = ecs.Transform{Scale: emath.Vec3{X: 1, Y: 1, Z: 1}}
			}
			skinned := isMeshSkinned(resolver, mr.MeshAssetID)
			var vb *wgpu.Buffer
			var vc uint32
			if skinned {
				vb, vc = sc.getOrCreateSkinnedMeshBuffer(resolver, mr.MeshAssetID)
				if sc.shadowSkinnedPipeline != nil {
					spotPass.SetPipeline(sc.shadowSkinnedPipeline)
					if !skinnedSet {
						spotPass.SetBindGroup(0, sc.shadowSkinnedBindGroup, nil)
						skinnedSet = true
					}
					if anim, ok := frame.World.GetAnimator(id); ok {
						writeBoneMatrices(boneBytes, anim.BoneMatrices)
					} else {
						writeBoneMatrices(boneBytes, nil)
					}
					s.queue.WriteBuffer(sc.boneUniform, 0, boneBytes)
				}
			} else {
				vb, vc = sc.getOrCreateMeshBuffer(resolver, mr.MeshAssetID)
				spotPass.SetPipeline(sc.shadowPipeline)
				if skinnedSet {
					spotPass.SetBindGroup(0, sc.shadowBindGroup, nil)
					skinnedSet = false
				}
			}
			if vb == nil {
				continue
			}
			model := buildModelMatrix(&tr)
			copy(shadowUbBytes[64:128], matrixToBytes(model))
			if skinned && sc.shadowSkinnedUniform != nil {
				s.queue.WriteBuffer(sc.shadowSkinnedUniform, 0, shadowUbBytes)
			} else {
				s.queue.WriteBuffer(sc.shadowUniform, 0, shadowUbBytes)
			}
			spotPass.SetVertexBuffer(0, vb, 0, wgpu.WholeSize)
			spotPass.Draw(vc, 1, 0, 0)
		}
		spotPass.End()
		spotPass.Release()
	}

	// Main pass (GPU instancing: batch by mesh+material)
	// Рендер в multisample-таргет (MSAA 4×, HDR RGBA16Float) с depth-буфером,
	// резолв в offscreen HDR-текстуру сцены; пост-процесс — в surface.
	if err := sc.ensureMSAA(s.device, wgpu.TextureFormatRGBA16Float, width, height, 4); err != nil {
		return err
	}
	if err := sc.ensurePostTargets(s.device, width, height); err != nil {
		return err
	}
	renderPass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View:          sc.msaaView,
				ResolveTarget: sc.sceneView,
				LoadOp:        wgpu.LoadOpClear,
				StoreOp:       wgpu.StoreOpStore,
				ClearValue:    cc,
			},
		},
		DepthStencilAttachment: &wgpu.RenderPassDepthStencilAttachment{
			View:            sc.msaaDepthView,
			DepthLoadOp:     wgpu.LoadOpClear,
			DepthStoreOp:    wgpu.StoreOpStore,
			DepthClearValue: 1.0,
		},
	})
	renderPass.SetPipeline(sc.pipeline)
	renderPass.SetBindGroup(0, sc.bindGroup, nil)
	renderPass.SetBindGroup(1, sc.bindGroupShadow, nil)
	if sc.envBindGroup != nil {
		renderPass.SetBindGroup(3, sc.envBindGroup, nil)
	}

	camPos := emath.Vec3{}
	if len(pbrData.camPos) == 3 {
		camPos = emath.Vec3{X: pbrData.camPos[0], Y: pbrData.camPos[1], Z: pbrData.camPos[2]}
	}
	batches := buildInstanceBatches(frame.World, &frustum, camPos, resolver)
	for _, batch := range batches {
		mat := batch.material
		if mat == nil {
			mat = getMeshMaterial(nil, nil)
		}
		writePBRUniforms(ubBytes, pbrData.viewProj, mat.baseColor, mat.metallic, mat.roughness,
			pbrData.lightDir, pbrData.lightIntensity, pbrData.lightColor,
			pbrData.ambient, pbrData.camPos, lightViewProj, nil,
			mat.emissiveColor, mat.emissiveStrength, mat.normalScale, mat.alphaCutoff, mat.flags,
			pbrData.pointLightPos, pbrData.pointLightIntensity, pbrData.pointLightColor, pbrData.pointLightRange,
			pbrData.spotLightPos, pbrData.spotLightDir, pbrData.spotLightColor, pbrData.spotLightIntensity, pbrData.spotLightRange, pbrData.spotInnerCos, pbrData.spotOuterCos,
			pbrData.spotLightViewProj)
		s.queue.WriteBuffer(sc.uniformBuffer, 0, ubBytes)

		mg := sc.getMaterialBindGroup(resolver, mat)
		if mg == nil {
			continue // ресурсы не готовы — пропуск, иначе draw упадёт по валидации
		}
		renderPass.SetBindGroup(2, mg, nil)

		vb, vc := sc.getOrCreateMeshBuffer(resolver, batch.meshAssetID)
		if vb == nil {
			continue
		}
		instanceBuf := buildInstanceBuffer(s.device, batch.transforms)
		if instanceBuf == nil {
			continue
		}
		renderPass.SetVertexBuffer(0, vb, 0, wgpu.WholeSize)
		renderPass.SetVertexBuffer(1, instanceBuf, 0, wgpu.WholeSize)
		renderPass.Draw(vc, uint32(len(batch.transforms)), 0, 0)
		transientBuffers = append(transientBuffers, instanceBuf)
	}

	// Skinned mesh pass (bone matrices в vertex shader)
	skinnedDraws := buildSkinnedDraws(frame.World, &frustum, camPos, resolver)
	if len(skinnedDraws) > 0 && sc.skinnedPipeline != nil {
		renderPass.SetPipeline(sc.skinnedPipeline)
		renderPass.SetBindGroup(0, sc.skinnedBindGroup, nil)
		renderPass.SetBindGroup(1, sc.skinnedBindGroupShadow, nil)
		if sc.envBindGroup != nil {
			renderPass.SetBindGroup(3, sc.envBindGroup, nil)
		}
		for _, d := range skinnedDraws {
			vb, vc := sc.getOrCreateSkinnedMeshBuffer(resolver, d.meshAssetID)
			if vb == nil {
				continue
			}
			mat := d.material
			if mat == nil {
				mat = getMeshMaterial(nil, nil)
			}
			model := buildModelMatrix(&d.transform)
			writePBRUniforms(ubBytes, pbrData.viewProj, mat.baseColor, mat.metallic, mat.roughness,
				pbrData.lightDir, pbrData.lightIntensity, pbrData.lightColor,
				pbrData.ambient, pbrData.camPos, lightViewProj, &model,
				mat.emissiveColor, mat.emissiveStrength, mat.normalScale, mat.alphaCutoff, mat.flags,
				pbrData.pointLightPos, pbrData.pointLightIntensity, pbrData.pointLightColor, pbrData.pointLightRange,
				pbrData.spotLightPos, pbrData.spotLightDir, pbrData.spotLightColor, pbrData.spotLightIntensity, pbrData.spotLightRange, pbrData.spotInnerCos, pbrData.spotOuterCos,
				pbrData.spotLightViewProj)
			s.queue.WriteBuffer(sc.skinnedUniform, 0, ubBytes)
			// Bone matrices сущности (без Animator — bind pose)
			var matrices []float32
			if anim, ok := frame.World.GetAnimator(d.entityID); ok {
				matrices = anim.BoneMatrices
			}
			writeBoneMatrices(boneBytes, matrices)
			s.queue.WriteBuffer(sc.boneUniform, 0, boneBytes)
			mg := sc.getMaterialBindGroup(resolver, mat)
			if mg == nil {
				continue
			}
			renderPass.SetBindGroup(2, mg, nil)
			renderPass.SetVertexBuffer(0, vb, 0, wgpu.WholeSize)
			renderPass.Draw(vc, 1, 0, 0)
		}
		renderPass.SetPipeline(sc.pipeline)
		renderPass.SetBindGroup(0, sc.bindGroup, nil)
		renderPass.SetBindGroup(1, sc.bindGroupShadow, nil)
		if sc.envBindGroup != nil {
			renderPass.SetBindGroup(3, sc.envBindGroup, nil)
		}
	}

	// Fallback: cube if no meshes
	if frame.World != nil {
		hasMesh := false
		for _, id := range frame.World.Entities() {
			if _, ok := frame.World.GetMeshRenderer(id); ok {
				hasMesh = true
				break
			}
		}
		if !hasMesh {
			tr := ecs.Transform{Position: emath.Vec3{X: 0, Y: 0, Z: 0}, Scale: emath.Vec3{X: 1, Y: 1, Z: 1}}
			writePBRUniforms(ubBytes, pbrData.viewProj, []float32{0.75, 0.75, 0.78}, 0.0, 0.5,
				pbrData.lightDir, pbrData.lightIntensity, pbrData.lightColor,
				pbrData.ambient, pbrData.camPos, lightViewProj, nil,
				[]float32{0, 0, 0}, 1.0, 1.0, 0.5, 0,
				pbrData.pointLightPos, pbrData.pointLightIntensity, pbrData.pointLightColor, pbrData.pointLightRange,
				pbrData.spotLightPos, pbrData.spotLightDir, pbrData.spotLightColor, pbrData.spotLightIntensity, pbrData.spotLightRange, pbrData.spotInnerCos, pbrData.spotOuterCos,
				pbrData.spotLightViewProj)
			s.queue.WriteBuffer(sc.uniformBuffer, 0, ubBytes)
			mg := sc.getMaterialBindGroup(resolver, nil)
			if mg != nil {
				renderPass.SetBindGroup(2, mg, nil)
			}
			instanceBuf := buildInstanceBuffer(s.device, []ecs.Transform{tr})
			if instanceBuf != nil {
				renderPass.SetVertexBuffer(0, sc.cubeVertexBuf, 0, wgpu.WholeSize)
				renderPass.SetVertexBuffer(1, instanceBuf, 0, wgpu.WholeSize)
				renderPass.Draw(sc.cubeVertexCount, 1, 0, 0)
				transientBuffers = append(transientBuffers, instanceBuf)
			}
		}
	}

	renderPass.End()
	renderPass.Release()

	// --- Пост-процесс: bright -> blur (2×) -> composite в surface ---
	if sc.compositePipeline != nil && sc.brightBG != nil && sc.blurAB != nil && sc.compositeBG != nil {
		bw, bh := width/2, height/2
		if bw < 1 {
			bw = 1
		}
		if bh < 1 {
			bh = 1
		}
		postUbBytes := make([]byte, 32)

		brightPass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
			ColorAttachments: []wgpu.RenderPassColorAttachment{
				{View: sc.bloomViewA, LoadOp: wgpu.LoadOpClear, StoreOp: wgpu.StoreOpStore, ClearValue: wgpu.Color{}},
			},
		})
		brightPass.SetPipeline(sc.brightPipeline)
		brightPass.SetBindGroup(0, sc.brightBG, nil)
		brightPass.Draw(3, 1, 0, 0)
		brightPass.End()
		brightPass.Release()

		// Горизонтальный блюр A -> B.
		binary.LittleEndian.PutUint32(postUbBytes[0:], math.Float32bits(1.0/float32(bw)))
		binary.LittleEndian.PutUint32(postUbBytes[4:], math.Float32bits(0))
		s.queue.WriteBuffer(sc.postUniform, 0, postUbBytes)
		blurH := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
			ColorAttachments: []wgpu.RenderPassColorAttachment{
				{View: sc.bloomViewB, LoadOp: wgpu.LoadOpClear, StoreOp: wgpu.StoreOpStore, ClearValue: wgpu.Color{}},
			},
		})
		blurH.SetPipeline(sc.blurPipeline)
		blurH.SetBindGroup(0, sc.blurAB, nil)
		blurH.Draw(3, 1, 0, 0)
		blurH.End()
		blurH.Release()

		// Вертикальный блюр B -> A.
		binary.LittleEndian.PutUint32(postUbBytes[0:], math.Float32bits(0))
		binary.LittleEndian.PutUint32(postUbBytes[4:], math.Float32bits(1.0/float32(bh)))
		s.queue.WriteBuffer(sc.postUniform, 0, postUbBytes)
		blurV := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
			ColorAttachments: []wgpu.RenderPassColorAttachment{
				{View: sc.bloomViewA, LoadOp: wgpu.LoadOpClear, StoreOp: wgpu.StoreOpStore, ClearValue: wgpu.Color{}},
			},
		})
		blurV.SetPipeline(sc.blurPipeline)
		blurV.SetBindGroup(0, sc.blurBA, nil)
		blurV.Draw(3, 1, 0, 0)
		blurV.End()
		blurV.Release()

		// Композит: ACES-тонмаппинг + bloom + виньетка -> swapchain.
		binary.LittleEndian.PutUint32(postUbBytes[0:], math.Float32bits(0))
		binary.LittleEndian.PutUint32(postUbBytes[4:], math.Float32bits(0))
		binary.LittleEndian.PutUint32(postUbBytes[8:], math.Float32bits(1.0))   // bloom intensity
		binary.LittleEndian.PutUint32(postUbBytes[12:], math.Float32bits(0.35)) // vignette
		binary.LittleEndian.PutUint32(postUbBytes[16:], math.Float32bits(0.6))  // dof strength
		binary.LittleEndian.PutUint32(postUbBytes[20:], math.Float32bits(0.5))  // focus y
		binary.LittleEndian.PutUint32(postUbBytes[24:], math.Float32bits(0.3))  // focus range
		s.queue.WriteBuffer(sc.postUniform, 0, postUbBytes)
		compositePass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
			ColorAttachments: []wgpu.RenderPassColorAttachment{
				{View: view, LoadOp: wgpu.LoadOpClear, StoreOp: wgpu.StoreOpStore, ClearValue: cc},
			},
		})
		compositePass.SetPipeline(sc.compositePipeline)
		compositePass.SetBindGroup(0, sc.compositeBG, nil)
		compositePass.Draw(3, 1, 0, 0)
		compositePass.End()
		compositePass.Release()
	}

	// --- HUD-оверлей поверх кадра (полосы здоровья, текст, экраны победы/поражения) ---
	if frame.HUD != nil {
		if err := sc.renderHUD(encoder, view, frame.HUD, width, height, s.config.Format); err != nil {
			// Ошибка HUD не должна убивать кадр — сцена уже отрисована
			fmt.Fprintf(os.Stderr, "kenga: hud: %v\n", err)
		}
	}

	cmdBuffer, err := encoder.Finish(nil)
	if err != nil {
		return err
	}
	defer cmdBuffer.Release()

	s.queue.Submit(cmdBuffer)
	for _, b := range transientBuffers {
		b.Release()
	}
	s.surface.Present()
	return nil
}

func (s *state) Destroy() {
	if s.scene != nil {
		s.scene.destroyHUD()
		for _, c := range s.scene.meshCache {
			if c != nil && c.vertexBuf != nil {
				c.vertexBuf.Release()
			}
		}
		s.scene.meshCache = nil
		for _, c := range s.scene.meshSkinnedCache {
			if c != nil && c.vertexBuf != nil {
				c.vertexBuf.Release()
			}
		}
		s.scene.meshSkinnedCache = nil
		if s.scene.shadowSkinnedPipeline != nil {
			s.scene.shadowSkinnedPipeline.Release()
		}
		if s.scene.shadowSkinnedBindGroup != nil {
			s.scene.shadowSkinnedBindGroup.Release()
		}
		if s.scene.shadowSkinnedUniform != nil {
			s.scene.shadowSkinnedUniform.Release()
		}
		if s.scene.skinnedBindGroupShadow != nil {
			s.scene.skinnedBindGroupShadow.Release()
		}
		if s.scene.skinnedBindGroup != nil {
			s.scene.skinnedBindGroup.Release()
		}
		if s.scene.boneUniform != nil {
			s.scene.boneUniform.Release()
		}
		if s.scene.skinnedUniform != nil {
			s.scene.skinnedUniform.Release()
		}
		if s.scene.skinnedPipeline != nil {
			s.scene.skinnedPipeline.Release()
		}
		if s.scene.bindGroupShadow != nil {
			s.scene.bindGroupShadow.Release()
		}
		if s.scene.bindGroup != nil {
			s.scene.bindGroup.Release()
		}
		if s.scene.shadowBindGroup != nil {
			s.scene.shadowBindGroup.Release()
		}
		if s.scene.shadowSampler != nil {
			s.scene.shadowSampler.Release()
		}
		if s.scene.shadowView != nil {
			s.scene.shadowView.Release()
		}
		if s.scene.pointShadowBindGroup != nil {
			s.scene.pointShadowBindGroup.Release()
		}
		if s.scene.pointShadowPipeline != nil {
			s.scene.pointShadowPipeline.Release()
		}
		if s.scene.pointShadowUniform != nil {
			s.scene.pointShadowUniform.Release()
		}
		for i := 0; i < 6; i++ {
			if s.scene.pointShadowFaces[i] != nil {
				s.scene.pointShadowFaces[i].Release()
			}
		}
		if s.scene.pointShadowView != nil {
			s.scene.pointShadowView.Release()
		}
		if s.scene.pointShadowMap != nil {
			s.scene.pointShadowMap.Release()
		}
		if s.scene.pointShadowSkinnedBindGroup != nil {
			s.scene.pointShadowSkinnedBindGroup.Release()
		}
		if s.scene.pointShadowSkinnedPipeline != nil {
			s.scene.pointShadowSkinnedPipeline.Release()
		}
		if s.scene.pointShadowSkinnedUniform != nil {
			s.scene.pointShadowSkinnedUniform.Release()
		}
		if s.scene.shadowMap != nil {
			s.scene.shadowMap.Release()
		}
		if s.scene.spotShadowView != nil {
			s.scene.spotShadowView.Release()
		}
		if s.scene.spotShadowMap != nil {
			s.scene.spotShadowMap.Release()
		}
		if s.scene.shadowPipeline != nil {
			s.scene.shadowPipeline.Release()
		}
		if s.scene.shadowUniform != nil {
			s.scene.shadowUniform.Release()
		}
		if s.scene.pipeline != nil {
			s.scene.pipeline.Release()
		}
		s.scene.releaseMSAA()
		if s.scene.uniformBuffer != nil {
			s.scene.uniformBuffer.Release()
		}
		if s.scene.cubeVertexBuf != nil {
			s.scene.cubeVertexBuf.Release()
		}
		for _, t := range s.scene.textureCache {
			if t != nil {
				t.Release()
			}
		}
		s.scene.textureCache = nil
		for _, bg := range s.scene.materialGroups {
			if bg != nil {
				bg.Release()
			}
		}
		s.scene.materialGroups = nil
		if s.scene.defaultMaterialGroup != nil {
			s.scene.defaultMaterialGroup.Release()
		}
		if s.scene.envBindGroup != nil {
			s.scene.envBindGroup.Release()
		}
		if s.scene.envSampler != nil {
			s.scene.envSampler.Release()
		}
		if s.scene.irrView != nil {
			s.scene.irrView.Release()
		}
		if s.scene.irrTex != nil {
			s.scene.irrTex.Release()
		}
		if s.scene.envView != nil {
			s.scene.envView.Release()
		}
		if s.scene.envTex != nil {
			s.scene.envTex.Release()
		}
		for _, bg := range []*wgpu.BindGroup{s.scene.compositeBG, s.scene.blurBA, s.scene.blurAB, s.scene.brightBG} {
			if bg != nil {
				bg.Release()
			}
		}
		for _, bgl := range []*wgpu.BindGroupLayout{s.scene.postBGLBright, s.scene.postBGLBlur, s.scene.postBGLComposite} {
			if bgl != nil {
				bgl.Release()
			}
		}
		if s.scene.postUniform != nil {
			s.scene.postUniform.Release()
		}
		if s.scene.compositePipeline != nil {
			s.scene.compositePipeline.Release()
		}
		if s.scene.blurPipeline != nil {
			s.scene.blurPipeline.Release()
		}
		if s.scene.brightPipeline != nil {
			s.scene.brightPipeline.Release()
		}
		s.scene.releasePost()
		if s.scene.textureSampler != nil {
			s.scene.textureSampler.Release()
		}
		if s.scene.fallbackBlack != nil {
			s.scene.fallbackBlack.Release()
		}
		if s.scene.fallbackNormal != nil {
			s.scene.fallbackNormal.Release()
		}
		if s.scene.fallbackWhite != nil {
			s.scene.fallbackWhite.Release()
		}
		s.scene = nil
	}
	if s.pipeline != nil {
		s.pipeline.Release()
		s.pipeline = nil
	}
	s.config = nil
	if s.queue != nil {
		s.queue.Release()
		s.queue = nil
	}
	if s.device != nil {
		s.device.Release()
		s.device = nil
	}
	if s.surface != nil {
		s.surface.Release()
		s.surface = nil
	}
	if s.instance != nil {
		s.instance.Release()
		s.instance = nil
	}
}
