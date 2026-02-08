//go:build webgpu && !js

package webgpu

import (
	_ "embed"
	"encoding/binary"
	"math"

	"github.com/cogentcore/webgpu/wgpu"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/ecs"
	"goenginekenga/engine/render"
	emath "goenginekenga/engine/math"
)

//go:embed shadow.wgsl
var shadowWGSL string

// buildModelMatrix строит матрицу модели из Transform (как в ebiten renderer).
func buildModelMatrix(tr *ecs.Transform) render.Matrix4 {
	t := render.Translate(tr.Position)
	rx := render.RotateX(tr.Rotation.X)
	ry := render.RotateY(tr.Rotation.Y)
	rz := render.RotateZ(tr.Rotation.Z)
	scale := tr.Scale
	if scale.X == 0 {
		scale.X = 1
	}
	if scale.Y == 0 {
		scale.Y = 1
	}
	if scale.Z == 0 {
		scale.Z = 1
	}
	s := render.Scale(scale)
	return t.Multiply(rz.Multiply(ry.Multiply(rx.Multiply(s))))
}

// matrixToBytes transposes row-major Matrix4 to column-major for WGSL.
func matrixToBytes(m render.Matrix4) []byte {
	out := make([]byte, 64)
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			binary.LittleEndian.PutUint32(out[(col*4+row)*4:], math.Float32bits(m[row*4+col]))
		}
	}
	return out
}

// pbrUniformsSize — размер uniform buffer для PBR + light_view_proj.
const pbrUniformsSize = 320

// writePBRUniforms пишет viewProj, material, light, camera, light_view_proj в буфер.
// model передаётся через instance buffer.
func writePBRUniforms(out []byte, viewProj render.Matrix4, baseColor []float32, metallic, roughness float32,
	lightDir []float32, lightIntensity float32, lightColor []float32, ambient float32, camPos []float32, lightViewProj render.Matrix4) {
	if len(out) < pbrUniformsSize {
		return
	}
	putF32 := func(off int, v float32) {
		binary.LittleEndian.PutUint32(out[off:], math.Float32bits(v))
	}
	putVec3 := func(off int, v []float32) {
		for i := 0; i < 3 && i < len(v); i++ {
			putF32(off+i*4, v[i])
		}
	}

	copy(out[0:64], matrixToBytes(viewProj))

	bc := baseColor
	if len(bc) < 3 {
		bc = []float32{0.8, 0.8, 0.8}
	}
	putVec3(128, bc)
	putF32(144, metallic)
	putF32(148, roughness)

	ld := lightDir
	if len(ld) < 3 {
		ld = []float32{0.5, 1.0, 0.3}
	}
	putVec3(160, ld)
	putF32(176, lightIntensity)

	lc := lightColor
	if len(lc) < 3 {
		lc = []float32{1.0, 1.0, 1.0}
	}
	putVec3(192, lc)
	putF32(208, ambient)

	cp := camPos
	if len(cp) < 3 {
		cp = []float32{0, 0, 5}
	}
	putVec3(224, cp)

	copy(out[256:320], matrixToBytes(lightViewProj))
}

// buildVertexData создаёт interleaved vertex buffer: pos(12) + normal(12) + uv(8) = 32 bytes/vertex.
func buildVertexData(positions, normals, uvs []float32, indices []uint32) []byte {
	if len(positions) == 0 || len(indices) == 0 {
		return nil
	}
	// Expand indexed to non-indexed for simplicity (or use index buffer)
	// For indexed: we need to create vertex buffer with all vertices, then index buffer
	// For non-indexed: expand triangles
	vertexCount := len(indices)
	stride := 32
	data := make([]byte, vertexCount*stride)
	for i := 0; i < vertexCount; i++ {
		idx := int(indices[i])
		off := i * stride
		// position
		if idx*3+2 < len(positions) {
			binary.LittleEndian.PutUint32(data[off:], math.Float32bits(positions[idx*3]))
			binary.LittleEndian.PutUint32(data[off+4:], math.Float32bits(positions[idx*3+1]))
			binary.LittleEndian.PutUint32(data[off+8:], math.Float32bits(positions[idx*3+2]))
		}
		// normal
		if len(normals) > 0 && idx*3+2 < len(normals) {
			binary.LittleEndian.PutUint32(data[off+12:], math.Float32bits(normals[idx*3]))
			binary.LittleEndian.PutUint32(data[off+16:], math.Float32bits(normals[idx*3+1]))
			binary.LittleEndian.PutUint32(data[off+20:], math.Float32bits(normals[idx*3+2]))
		} else {
			binary.LittleEndian.PutUint32(data[off+12:], math.Float32bits(0))
			binary.LittleEndian.PutUint32(data[off+16:], math.Float32bits(1))
			binary.LittleEndian.PutUint32(data[off+20:], math.Float32bits(0))
		}
		// uv
		if len(uvs) > 0 && idx*2+1 < len(uvs) {
			binary.LittleEndian.PutUint32(data[off+24:], math.Float32bits(uvs[idx*2]))
			binary.LittleEndian.PutUint32(data[off+28:], math.Float32bits(uvs[idx*2+1]))
		}
	}
	return data
}

// defaultCubeVertexData returns vertex data for a unit cube (fallback).
func defaultCubeVertexData() []byte {
	cube := render.CreateCube()
	return buildVertexData(cube.Vertices, cube.Normals, cube.UVs, cube.Indices)
}

// meshFromResolver загружает меш из resolver или возвращает cube.
func meshFromResolver(resolver *asset.Resolver, meshAssetID string) (positions, normals, uvs []float32, indices []uint32) {
	if resolver != nil && meshAssetID != "" {
		if mesh, err := resolver.ResolveMeshByAssetID(meshAssetID); err == nil {
			return mesh.Positions, mesh.Normals, mesh.UV0, mesh.Indices
		}
	}
	cube := render.CreateCube()
	return cube.Vertices, cube.Normals, cube.UVs, cube.Indices
}

// sceneState holds GPU resources for rendering a 3D scene.
type sceneState struct {
	device          *wgpu.Device
	queue           *wgpu.Queue
	pipeline        *wgpu.RenderPipeline
	uniformBuffer   *wgpu.Buffer
	bindGroup       *wgpu.BindGroup
	bindGroupShadow *wgpu.BindGroup
	cubeVertexBuf   *wgpu.Buffer
	cubeVertexCount uint32

	shadowMap       *wgpu.Texture
	shadowView      *wgpu.TextureView
	shadowPipeline  *wgpu.RenderPipeline
	shadowUniform   *wgpu.Buffer
	shadowBindGroup *wgpu.BindGroup
	shadowSampler   *wgpu.Sampler
}

func (s *state) initSceneState() error {
	// Uniform buffer: PBR struct (MVP, model, material, light, camera)
	ub, err := s.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "uniforms",
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
		Size:  pbrUniformsSize,
	})
	if err != nil {
		return err
	}

	// Default cube vertex buffer
	cubeData := defaultCubeVertexData()
	cubeBuf, err := s.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "cube vertices",
		Contents: cubeData,
		Usage:    wgpu.BufferUsageVertex,
	})
	if err != nil {
		ub.Release()
		return err
	}

	shader, err := s.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:          "mesh shader",
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{Code: shaderWGSL},
	})
	if err != nil {
		cubeBuf.Release()
		ub.Release()
		return err
	}

	// Pipeline: layout inferred from shader
	pipeline, err := s.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "Mesh Pipeline",
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{
				{
					ArrayStride: 32,
					StepMode:    wgpu.VertexStepModeVertex,
					Attributes: []wgpu.VertexAttribute{
						{Format: wgpu.VertexFormatFloat32x3, Offset: 0, ShaderLocation: 0},
						{Format: wgpu.VertexFormatFloat32x3, Offset: 12, ShaderLocation: 1},
						{Format: wgpu.VertexFormatFloat32x2, Offset: 24, ShaderLocation: 2},
					},
				},
				{
					ArrayStride: 64,
					StepMode:    wgpu.VertexStepModeInstance,
					Attributes: []wgpu.VertexAttribute{
						{Format: wgpu.VertexFormatFloat32x4, Offset: 0, ShaderLocation: 3},
						{Format: wgpu.VertexFormatFloat32x4, Offset: 16, ShaderLocation: 4},
						{Format: wgpu.VertexFormatFloat32x4, Offset: 32, ShaderLocation: 5},
						{Format: wgpu.VertexFormatFloat32x4, Offset: 48, ShaderLocation: 6},
					},
				},
			},
		},
		Primitive: wgpu.PrimitiveState{
			Topology:         wgpu.PrimitiveTopologyTriangleList,
			StripIndexFormat: wgpu.IndexFormatUndefined,
			FrontFace:        wgpu.FrontFaceCCW,
			CullMode:         wgpu.CullModeBack,
		},
		Multisample: wgpu.MultisampleState{Count: 1, Mask: 0xFFFFFFFF},
		Fragment: &wgpu.FragmentState{
			Module:     shader,
			EntryPoint: "fs_main",
			Targets: []wgpu.ColorTargetState{
				{Format: s.config.Format, Blend: &wgpu.BlendStateReplace, WriteMask: wgpu.ColorWriteMaskAll},
			},
		},
	})
	shader.Release()
	if err != nil {
		cubeBuf.Release()
		ub.Release()
		return err
	}

	bgl := pipeline.GetBindGroupLayout(0)
	bg, err := s.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout: bgl,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: ub, Size: pbrUniformsSize},
		},
	})
	bgl.Release()
	if err != nil {
		pipeline.Release()
		cubeBuf.Release()
		ub.Release()
		return err
	}

	// Shadow map: 2048x2048 depth texture
	shadowTex, err := s.device.CreateTexture(&wgpu.TextureDescriptor{
		Label:       "shadow map",
		Size:        wgpu.Extent3D{Width: 2048, Height: 2048, DepthOrArrayLayers: 1},
		MipLevelCount: 1,
		SampleCount: 1,
		Dimension:   wgpu.TextureDimension2D,
		Format:      wgpu.TextureFormatDepth32Float,
		Usage:       wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageTextureBinding,
	})
	if err != nil {
		pipeline.Release()
		cubeBuf.Release()
		ub.Release()
		bg.Release()
		return err
	}
	shadowView, err := shadowTex.CreateView(nil)
	if err != nil {
		shadowTex.Release()
		pipeline.Release()
		cubeBuf.Release()
		ub.Release()
		bg.Release()
		return err
	}

	shadowSampler, err := s.device.CreateSampler(&wgpu.SamplerDescriptor{
		Compare:     wgpu.CompareFunctionLessEqual,
		AddressModeU: wgpu.AddressModeClampToEdge,
		AddressModeV: wgpu.AddressModeClampToEdge,
	})
	if err != nil {
		shadowView.Release()
		shadowTex.Release()
		pipeline.Release()
		cubeBuf.Release()
		ub.Release()
		bg.Release()
		return err
	}

	shadowShader, err := s.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:          "shadow shader",
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{Code: shadowWGSL},
	})
	if err != nil {
		shadowSampler.Release()
		shadowView.Release()
		shadowTex.Release()
		pipeline.Release()
		cubeBuf.Release()
		ub.Release()
		bg.Release()
		return err
	}

	shadowUb, err := s.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "shadow uniforms",
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
		Size:  128,
	})
	if err != nil {
		shadowShader.Release()
		shadowSampler.Release()
		shadowView.Release()
		shadowTex.Release()
		pipeline.Release()
		cubeBuf.Release()
		ub.Release()
		bg.Release()
		return err
	}

	shadowPl, err := s.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "Shadow Pipeline",
		Vertex: wgpu.VertexState{
			Module:     shadowShader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{
				{ArrayStride: 32, StepMode: wgpu.VertexStepModeVertex,
					Attributes: []wgpu.VertexAttribute{
						{Format: wgpu.VertexFormatFloat32x3, Offset: 0, ShaderLocation: 0},
						{Format: wgpu.VertexFormatFloat32x3, Offset: 12, ShaderLocation: 1},
						{Format: wgpu.VertexFormatFloat32x2, Offset: 24, ShaderLocation: 2},
					},
				},
			},
		},
		Primitive: wgpu.PrimitiveState{
			Topology: wgpu.PrimitiveTopologyTriangleList,
			FrontFace: wgpu.FrontFaceCCW,
			CullMode:  wgpu.CullModeBack,
		},
		DepthStencil: &wgpu.DepthStencilState{
			Format:            wgpu.TextureFormatDepth32Float,
			DepthWriteEnabled: true,
			DepthCompare:      wgpu.CompareFunctionLessEqual,
		},
		Fragment: nil,
	})
	shadowShader.Release()
	if err != nil {
		shadowUb.Release()
		shadowSampler.Release()
		shadowView.Release()
		shadowTex.Release()
		pipeline.Release()
		cubeBuf.Release()
		ub.Release()
		bg.Release()
		return err
	}

	sbgl := shadowPl.GetBindGroupLayout(0)
	sbg, err := s.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout: sbgl,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: shadowUb, Size: 128},
		},
	})
	sbgl.Release()
	if err != nil {
		shadowPl.Release()
		shadowUb.Release()
		shadowSampler.Release()
		shadowView.Release()
		shadowTex.Release()
		pipeline.Release()
		cubeBuf.Release()
		ub.Release()
		bg.Release()
		return err
	}

	bgl1 := pipeline.GetBindGroupLayout(1)
	bgShadow, err := s.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout: bgl1,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, TextureView: shadowView},
			{Binding: 1, Sampler: shadowSampler},
		},
	})
	bgl1.Release()
	if err != nil {
		shadowPl.Release()
		shadowUb.Release()
		shadowSampler.Release()
		shadowView.Release()
		shadowTex.Release()
		pipeline.Release()
		cubeBuf.Release()
		ub.Release()
		bg.Release()
		return err
	}

	s.scene = &sceneState{
		device:          s.device,
		queue:           s.queue,
		pipeline:        pipeline,
		uniformBuffer:   ub,
		bindGroup:       bg,
		bindGroupShadow: bgShadow,
		cubeVertexBuf:   cubeBuf,
		cubeVertexCount: uint32(len(cubeData) / 32),
		shadowMap:       shadowTex,
		shadowView:      shadowView,
		shadowPipeline:  shadowPl,
		shadowUniform:   shadowUb,
		shadowBindGroup: sbg,
		shadowSampler:   shadowSampler,
	}
	return nil
}

// pbrSceneData содержит данные для PBR рендера из World.
type pbrSceneData struct {
	viewProj      render.Matrix4
	model         render.Matrix4
	camPos        []float32
	lightDir      []float32
	lightColor    []float32
	lightIntensity float32
	baseColor     []float32
	metallic      float32
	roughness     float32
	ambient       float32
}

// buildLightViewProj строит orthographic view-projection для directional light.
func buildLightViewProj(lightDir []float32) render.Matrix4 {
	ld := lightDir
	if len(ld) < 3 {
		ld = []float32{0.5, 1.0, 0.3}
	}
	dir := emath.Vec3{X: ld[0], Y: ld[1], Z: ld[2]}
	dir = render.Normalize3(dir)
	dist := float32(30)
	eye := emath.Vec3{X: -dir.X * dist, Y: -dir.Y * dist, Z: -dir.Z * dist}
	target := emath.Vec3{X: 0, Y: 0, Z: 0}
	up := emath.Vec3{X: 0, Y: 1, Z: 0}
	view := render.LookAt(eye, target, up)
	ortho := render.Orthographic(-20, 20, -20, 20, 0.1, 60)
	return ortho.Multiply(view)
}

// getPBRSceneData извлекает камеру, свет и параметры из World.
func getPBRSceneData(world *ecs.World, width, height int) (data pbrSceneData, ok bool) {
	if world == nil {
		return data, false
	}
	var camPos emath.Vec3
	for _, id := range world.Entities() {
		cam, hasCam := world.GetCamera(id)
		if !hasCam {
			continue
		}
		tr, hasTr := world.GetTransform(id)
		if !hasTr {
			tr = ecs.Transform{Position: emath.Vec3{X: 0, Y: 0, Z: 5}, Scale: emath.Vec3{X: 1, Y: 1, Z: 1}}
		}
		camPos = tr.Position
		radY := tr.Rotation.Y * math.Pi / 180
		forward := emath.Vec3{
			X: float32(math.Sin(float64(radY))),
			Y: 0,
			Z: float32(math.Cos(float64(radY))),
		}
		target := emath.Vec3{X: tr.Position.X + forward.X*10, Y: tr.Position.Y, Z: tr.Position.Z + forward.Z*10}
		c := render.NewCamera3D()
		c.SetPosition(tr.Position)
		c.SetTarget(target)
		if cam.FovYDegrees > 0 {
			c.SetFOV(cam.FovYDegrees)
		}
		c.SetAspectRatio(float32(width) / float32(height))
		data.viewProj = c.GetViewProjectionMatrix()
		data.camPos = []float32{camPos.X, camPos.Y, camPos.Z}
		break
	}
	if len(data.camPos) == 0 {
		c := render.NewCamera3D()
		c.SetPosition(emath.Vec3{X: 0, Y: 0, Z: 5})
		c.SetTarget(emath.Vec3{X: 0, Y: 0, Z: 0})
		c.SetAspectRatio(float32(width) / float32(height))
		data.viewProj = c.GetViewProjectionMatrix()
		data.camPos = []float32{0, 0, 5}
	}

	// Первый directional light
	for _, id := range world.Entities() {
		light, hasLight := world.GetLight(id)
		if !hasLight || light.Kind != "directional" {
			continue
		}
		tr, hasTr := world.GetTransform(id)
		dir := emath.Vec3{X: 0.5, Y: 1, Z: 0.3}
		if hasTr {
			radX := tr.Rotation.X * math.Pi / 180
			radY := tr.Rotation.Y * math.Pi / 180
			dir = emath.Vec3{
				X: float32(math.Sin(float64(radY)) * math.Cos(float64(radX))),
				Y: float32(-math.Sin(float64(radX))),
				Z: float32(math.Cos(float64(radY)) * math.Cos(float64(radX))),
			}
		}
		dir = render.Normalize3(dir)
		data.lightDir = []float32{dir.X, dir.Y, dir.Z}
		r, g, b := float32(light.ColorR)/255, float32(light.ColorG)/255, float32(light.ColorB)/255
		if r == 0 && g == 0 && b == 0 {
			r, g, b = light.ColorRGB.X, light.ColorRGB.Y, light.ColorRGB.Z
		}
		data.lightColor = []float32{r, g, b}
		data.lightIntensity = light.Intensity
		if data.lightIntensity <= 0 {
			data.lightIntensity = 1.0
		}
		break
	}
	if len(data.lightDir) == 0 {
		data.lightDir = []float32{0.5, 1.0, 0.3}
		data.lightColor = []float32{1.0, 1.0, 1.0}
		data.lightIntensity = 1.0
	}

	data.ambient = 0.15
	data.metallic = 0.0
	data.roughness = 0.5
	data.baseColor = []float32{0.8, 0.8, 0.8}
	return data, true
}

// instanceBatch — группа entities с одинаковым mesh и material для instancing.
type instanceBatch struct {
	meshAssetID   string
	materialKey   string // MaterialAssetID или "" для default
	transforms    []ecs.Transform
	baseColor     []float32
	metallic      float32
	roughness     float32
}

// buildInstanceBatches группирует entities по (mesh, material) для instancing.
func buildInstanceBatches(world *ecs.World, frustum *render.Frustum, resolver *asset.Resolver) []instanceBatch {
	group := make(map[string]*instanceBatch)

	for _, id := range world.Entities() {
		mr, hasMR := world.GetMeshRenderer(id)
		if !hasMR {
			continue
		}
		tr, hasTr := world.GetTransform(id)
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
		if frustum != nil && !frustum.SphereInFrustum(tr.Position, radius) {
			continue
		}
		key := mr.MeshAssetID + "|" + mr.MaterialAssetID
		if _, ok := group[key]; !ok {
			bc, metallic, roughness := getMeshMaterial(mr, resolver)
			group[key] = &instanceBatch{
				meshAssetID: mr.MeshAssetID,
				materialKey: mr.MaterialAssetID,
				transforms:  nil,
				baseColor:   bc,
				metallic:    metallic,
				roughness:   roughness,
			}
		}
		group[key].transforms = append(group[key].transforms, tr)
	}

	var batches []instanceBatch
	for _, b := range group {
		if len(b.transforms) > 0 {
			batches = append(batches, *b)
		}
	}
	return batches
}

// buildInstanceBuffer создаёт GPU buffer с model matrices для N instances.
func buildInstanceBuffer(device *wgpu.Device, transforms []ecs.Transform) *wgpu.Buffer {
	if len(transforms) == 0 {
		return nil
	}
	data := make([]byte, len(transforms)*64)
	for i, tr := range transforms {
		model := buildModelMatrix(&tr)
		copy(data[i*64:(i+1)*64], matrixToBytes(model))
	}
	buf, err := device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "instance matrices",
		Contents: data,
		Usage:    wgpu.BufferUsageVertex,
	})
	if err != nil {
		return nil
	}
	return buf
}

// getMeshMaterial возвращает baseColor, metallic, roughness из MeshRenderer и resolver.
func getMeshMaterial(mr *ecs.MeshRenderer, resolver *asset.Resolver) (baseColor []float32, metallic, roughness float32) {
	metallic = 0.0
	roughness = 0.5
	baseColor = []float32{0.8, 0.8, 0.8}

	if mr.ColorA > 0 {
		baseColor = []float32{
			float32(mr.ColorR) / 255,
			float32(mr.ColorG) / 255,
			float32(mr.ColorB) / 255,
		}
	} else if resolver != nil && mr.MaterialAssetID != "" {
		if mat, err := resolver.ResolveMaterialByAssetID(mr.MaterialAssetID); err == nil {
			baseColor = []float32{mat.BaseColor.X, mat.BaseColor.Y, mat.BaseColor.Z}
			metallic = mat.Metallic
			roughness = mat.Roughness
		}
	}

	return baseColor, metallic, roughness
}

// getCameraAndViewProj извлекает камеру из world и возвращает viewProj.
func getCameraAndViewProj(world *ecs.World, width, height int) (viewProj render.Matrix4, ok bool) {
	if world == nil {
		return render.Matrix4{}, false
	}
	for _, id := range world.Entities() {
		cam, hasCam := world.GetCamera(id)
		if !hasCam {
			continue
		}
		tr, hasTr := world.GetTransform(id)
		if !hasTr {
			tr = ecs.Transform{Position: emath.Vec3{X: 0, Y: 0, Z: 5}, Scale: emath.Vec3{X: 1, Y: 1, Z: 1}}
		}
		pos := tr.Position
		radY := tr.Rotation.Y * math.Pi / 180
		forward := emath.Vec3{
			X: float32(math.Sin(float64(radY))),
			Y: 0,
			Z: float32(math.Cos(float64(radY))),
		}
		target := emath.Vec3{X: pos.X + forward.X*10, Y: pos.Y, Z: pos.Z + forward.Z*10}
		c := render.NewCamera3D()
		c.SetPosition(pos)
		c.SetTarget(target)
		if cam.FovYDegrees > 0 {
			c.SetFOV(cam.FovYDegrees)
		}
		c.SetAspectRatio(float32(width) / float32(height))
		return c.GetViewProjectionMatrix(), true
	}
	// Fallback
	c := render.NewCamera3D()
	c.SetPosition(emath.Vec3{X: 0, Y: 0, Z: 5})
	c.SetTarget(emath.Vec3{X: 0, Y: 0, Z: 0})
	c.SetAspectRatio(float32(width) / float32(height))
	return c.GetViewProjectionMatrix(), true
}
