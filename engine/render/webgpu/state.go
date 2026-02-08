//go:build webgpu && !js

package webgpu

import (
	_ "embed"
	"math"

	"github.com/cogentcore/webgpu/wgpu"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/render"
)

//go:embed shader.wgsl
var shaderWGSL string

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
			vb, vc := sc.getOrCreateMeshBuffer(resolver, mr.MeshAssetID)
			if vb == nil {
				continue
			}
			model := buildModelMatrix(&tr)
			copy(shadowUbBytes[64:128], matrixToBytes(model))
			s.queue.WriteBuffer(sc.shadowUniform, 0, shadowUbBytes)
			shadowPass.SetVertexBuffer(0, vb, 0, wgpu.WholeSize)
			shadowPass.Draw(vc, 1, 0, 0)
		}
		shadowPass.End()
		shadowPass.Release()
	}

	// Main pass (GPU instancing: batch by mesh+material)
	renderPass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{View: view, LoadOp: wgpu.LoadOpClear, StoreOp: wgpu.StoreOpStore, ClearValue: cc},
		},
	})
	renderPass.SetPipeline(sc.pipeline)
	renderPass.SetBindGroup(0, sc.bindGroup, nil)
	renderPass.SetBindGroup(1, sc.bindGroupShadow, nil)

	batches := buildInstanceBatches(frame.World, &frustum, resolver)
	for _, batch := range batches {
		writePBRUniforms(ubBytes, pbrData.viewProj, batch.baseColor, batch.metallic, batch.roughness,
			pbrData.lightDir, pbrData.lightIntensity, pbrData.lightColor,
			pbrData.ambient, pbrData.camPos, lightViewProj)
		s.queue.WriteBuffer(sc.uniformBuffer, 0, ubBytes)

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
		instanceBuf.Release()
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
				pbrData.ambient, pbrData.camPos, lightViewProj)
			s.queue.WriteBuffer(sc.uniformBuffer, 0, ubBytes)
			instanceBuf := buildInstanceBuffer(s.device, []ecs.Transform{tr})
			if instanceBuf != nil {
				renderPass.SetVertexBuffer(0, sc.cubeVertexBuf, 0, wgpu.WholeSize)
				renderPass.SetVertexBuffer(1, instanceBuf, 0, wgpu.WholeSize)
				renderPass.Draw(sc.cubeVertexCount, 1, 0, 0)
				instanceBuf.Release()
			}
		}
	}

	renderPass.End()
	renderPass.Release()

	cmdBuffer, err := encoder.Finish(nil)
	if err != nil {
		return err
	}
	defer cmdBuffer.Release()

	s.queue.Submit(cmdBuffer)
	s.surface.Present()
	return nil
}

func (s *state) Destroy() {
	if s.scene != nil {
		for _, c := range s.scene.meshCache {
			if c != nil && c.vertexBuf != nil {
				c.vertexBuf.Release()
			}
		}
		s.scene.meshCache = nil
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
		if s.scene.shadowMap != nil {
			s.scene.shadowMap.Release()
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
		if s.scene.uniformBuffer != nil {
			s.scene.uniformBuffer.Release()
		}
		if s.scene.cubeVertexBuf != nil {
			s.scene.cubeVertexBuf.Release()
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
