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

//go:embed shadow.wgsl
var shadowWGSL string

//go:embed shadow_skinned.wgsl
var shadowSkinnedWGSL string

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

// pbrUniformsSize — размер uniform buffer для PBR + light_view_proj + emissive/texture params + spot light.
const pbrUniformsSize = 512

// writePBRUniforms пишет viewProj, material, light, camera, light_view_proj в буфер.
// model передаётся через instance buffer; для skeletal mesh — через modelInUniform (пишется в offset 64).
// Офсеты (bytes): 0 view_proj, 64 model, 128 base_color, 144 metallic, 148 roughness,
// 160 light_dir, 176 light_intensity, 192 light_color, 208 ambient, 224 cam_pos,
// 256 light_view_proj, 320 emissive_color, 336 emissive_strength, 340 normal_scale,
// 344 alpha_cutoff, 348 flags (bit0: есть metallicRoughness текстура),
// 352 point_light_pos + intensity, 368 point_light_color + range (intensity <= 0 = выключен),
// 384 spot_light_pos, 400 spot_light_dir, 416 spot_light_color, 432 intensity,
// 436 range, 440 inner_cos, 444 outer_cos (intensity <= 0 = выключен),
// 448 spot_light_view_proj (для spot-теней).
func writePBRUniforms(out []byte, viewProj render.Matrix4, baseColor []float32, metallic, roughness float32,
	lightDir []float32, lightIntensity float32, lightColor []float32, ambient float32, camPos []float32, lightViewProj render.Matrix4, modelInUniform *render.Matrix4,
	emissiveColor []float32, emissiveStrength, normalScale, alphaCutoff float32, flags uint32,
	pointLightPos []float32, pointLightIntensity float32, pointLightColor []float32, pointLightRange float32,
	spotLightPos []float32, spotLightDir []float32, spotLightColor []float32, spotLightIntensity, spotLightRange, spotInnerCos, spotOuterCos float32,
	spotLightViewProj render.Matrix4) {
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
	if modelInUniform != nil {
		copy(out[64:128], matrixToBytes(*modelInUniform))
	}

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

	ec := emissiveColor
	if len(ec) < 3 {
		ec = []float32{0, 0, 0}
	}
	putVec3(320, ec)
	putF32(336, emissiveStrength)
	putF32(340, normalScale)
	putF32(344, alphaCutoff)
	binary.LittleEndian.PutUint32(out[348:], flags)

	// Point light (offset 352): pos.xyz + intensity; интенсивность <= 0 отключает свет в шейдере.
	plp := pointLightPos
	if len(plp) < 3 {
		plp = []float32{0, 5, 0}
	}
	putVec3(352, plp)
	putF32(364, pointLightIntensity)
	plc := pointLightColor
	if len(plc) < 3 {
		plc = []float32{1.0, 1.0, 1.0}
	}
	putVec3(368, plc)
	putF32(380, pointLightRange)

	// Spotlight (offset 384): pos, dir, color, intensity/range/cos; интенсивность <= 0 выключает свет.
	slp := spotLightPos
	if len(slp) < 3 {
		slp = []float32{0, 5, 0}
	}
	putVec3(384, slp)
	sld := spotLightDir
	if len(sld) < 3 {
		sld = []float32{0, -1, 0}
	}
	putVec3(400, sld)
	slc := spotLightColor
	if len(slc) < 3 {
		slc = []float32{1.0, 1.0, 1.0}
	}
	putVec3(416, slc)
	putF32(432, spotLightIntensity)
	putF32(436, spotLightRange)
	putF32(440, spotInnerCos)
	putF32(444, spotOuterCos)
	copy(out[448:512], matrixToBytes(spotLightViewProj))
}

const boneMatrixCount = 64
const boneUniformsSize = boneMatrixCount * 64 // 64 mat4x4

var identityMatrix = render.Matrix4{
	1, 0, 0, 0,
	0, 1, 0, 0,
	0, 0, 1, 0,
	0, 0, 0, 1,
}

// writeBoneMatrices пишет bone matrices в uniform buffer (column-major, как в WGSL mat4x4).
// matrices — уже готовые column-major float32 (из ecs.Animator.BoneMatrices);
// кости сверх лимита и отсутствующие — identity (bind pose).
func writeBoneMatrices(out []byte, matrices []float32) {
	if len(out) < boneUniformsSize {
		return
	}
	identityBytes := matrixToBytes(identityMatrix)
	for i := 0; i < boneMatrixCount; i++ {
		if i*16+16 <= len(matrices) {
			for j := 0; j < 16; j++ {
				binary.LittleEndian.PutUint32(out[i*64+j*4:], math.Float32bits(matrices[i*16+j]))
			}
		} else {
			copy(out[i*64:(i+1)*64], identityBytes)
		}
	}
}

// ---------- Текстуры материалов ----------

// createTextureRGBA создаёт GPU-текстуру из RGBA-данных.
// Строки выравниваются до 256 байт (требование WriteTexture для bytesPerRow).
func (sc *sceneState) createTextureRGBA(label string, w, h int, data []byte, srgb bool) (*wgpu.Texture, error) {
	format := wgpu.TextureFormatRGBA8Unorm
	if srgb {
		format = wgpu.TextureFormatRGBA8UnormSrgb
	}
	tex, err := sc.device.CreateTexture(&wgpu.TextureDescriptor{
		Label:         label,
		Size:          wgpu.Extent3D{Width: uint32(w), Height: uint32(h), DepthOrArrayLayers: 1},
		MipLevelCount: 1,
		SampleCount:   1,
		Dimension:     wgpu.TextureDimension2D,
		Format:        format,
		Usage:         wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst,
	})
	if err != nil {
		return nil, err
	}
	rowBytes := w * 4
	aligned := ((rowBytes + 255) / 256) * 256
	upload := data
	if aligned != rowBytes {
		upload = make([]byte, aligned*h)
		for r := 0; r < h && (r+1)*rowBytes <= len(data); r++ {
			copy(upload[r*aligned:(r+1)*aligned], data[r*rowBytes:(r+1)*rowBytes])
		}
	}
	if err := sc.queue.WriteTexture(
		&wgpu.ImageCopyTexture{Texture: tex},
		upload,
		// RowsPerImage обязателен ненулевой: в wgpu-native это NonZeroU32 (conv.rs паникует на 0).
		&wgpu.TextureDataLayout{Offset: 0, BytesPerRow: uint32(aligned), RowsPerImage: uint32(h)},
		&wgpu.Extent3D{Width: uint32(w), Height: uint32(h), DepthOrArrayLayers: 1},
	); err != nil {
		tex.Release()
		return nil, err
	}
	return tex, nil
}

// ensureMaterialResources создаёт fallback-текстуры, сэмплер и default bind group (лениво).
func (sc *sceneState) ensureMaterialResources() {
	if sc.textureSampler != nil || sc.device == nil {
		return
	}
	white, err := sc.createTextureRGBA("fallback white", 1, 1, []byte{255, 255, 255, 255}, false)
	if err != nil {
		return
	}
	flat, err := sc.createTextureRGBA("fallback flat normal", 1, 1, []byte{128, 128, 255, 255}, false)
	if err != nil {
		white.Release()
		return
	}
	black, err := sc.createTextureRGBA("fallback black", 1, 1, []byte{0, 0, 0, 255}, false)
	if err != nil {
		white.Release()
		flat.Release()
		return
	}
	smp, err := sc.device.CreateSampler(&wgpu.SamplerDescriptor{
		Label:         "material sampler",
		AddressModeU:  wgpu.AddressModeRepeat,
		AddressModeV:  wgpu.AddressModeRepeat,
		AddressModeW:  wgpu.AddressModeRepeat,
		MagFilter:     wgpu.FilterModeLinear,
		MinFilter:     wgpu.FilterModeLinear,
		MipmapFilter:  wgpu.MipmapFilterModeLinear,
		MaxAnisotropy: 1,
	})
	if err != nil {
		white.Release()
		flat.Release()
		black.Release()
		return
	}
	sc.fallbackWhite = white
	sc.fallbackNormal = flat
	sc.fallbackBlack = black
	sc.textureSampler = smp
	sc.textureCache = make(map[string]*wgpu.Texture)
	sc.materialGroups = make(map[string]*wgpu.BindGroup)
	sc.defaultMaterialGroup = sc.makeMaterialBindGroup(nil, nil, nil, nil)
}

// getTexture возвращает GPU-текстуру для пути .texture.json (с кэшем);
// пустой путь или ошибка загрузки — fallback-текстура.
func (sc *sceneState) getTexture(resolver *asset.Resolver, path string, srgb bool, fallback *wgpu.Texture) *wgpu.Texture {
	sc.ensureMaterialResources()
	if fallback == nil {
		fallback = sc.fallbackWhite
	}
	if path == "" || sc.textureCache == nil || resolver == nil {
		return fallback
	}
	if t, ok := sc.textureCache[path]; ok {
		return t
	}
	tex, err := resolver.ResolveTextureByPath(path)
	if err != nil || tex == nil || len(tex.Data) < tex.Width*tex.Height*4 {
		return fallback
	}
	t, err := sc.createTextureRGBA("texture "+path, tex.Width, tex.Height, tex.Data, srgb)
	if err != nil {
		return fallback
	}
	sc.textureCache[path] = t
	return t
}

// makeMaterialBindGroup создаёт bind group (группа 2) из четырёх текстур и общего сэмплера.
// nil-текстуры заменяются fallback-текстурами.
func (sc *sceneState) makeMaterialBindGroup(texColor, texNormal, texMR, texEmissive *wgpu.Texture) *wgpu.BindGroup {
	if sc.pipeline == nil || sc.textureSampler == nil || sc.device == nil {
		return nil
	}
	if texColor == nil {
		texColor = sc.fallbackWhite
	}
	if texNormal == nil {
		texNormal = sc.fallbackNormal
	}
	if texMR == nil {
		texMR = sc.fallbackWhite
	}
	if texEmissive == nil {
		texEmissive = sc.fallbackBlack
	}

	views := make([]*wgpu.TextureView, 4)
	ok := true
	var err error
	for i, t := range []*wgpu.Texture{texColor, texNormal, texMR, texEmissive} {
		views[i], err = t.CreateView(nil)
		if err != nil {
			ok = false
			break
		}
	}
	if !ok {
		for _, v := range views {
			if v != nil {
				v.Release()
			}
		}
		return nil
	}
	defer func() {
		for _, v := range views {
			v.Release()
		}
	}()

	bgl := sc.pipeline.GetBindGroupLayout(2)
	bg, err := sc.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout: bgl,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, TextureView: views[0]},
			{Binding: 1, Sampler: sc.textureSampler},
			{Binding: 2, TextureView: views[1]},
			{Binding: 3, Sampler: sc.textureSampler},
			{Binding: 4, TextureView: views[2]},
			{Binding: 5, Sampler: sc.textureSampler},
			{Binding: 6, TextureView: views[3]},
			{Binding: 7, Sampler: sc.textureSampler},
		},
	})
	bgl.Release()
	if err != nil {
		return nil
	}
	return bg
}

// getMaterialBindGroup возвращает (и кэширует) bind group группы 2 для materialInfo.
// mat == nil — default bind group (все fallback-текстуры).
func (sc *sceneState) getMaterialBindGroup(resolver *asset.Resolver, mat *materialInfo) *wgpu.BindGroup {
	sc.ensureMaterialResources()
	if sc.materialGroups == nil {
		return nil
	}
	if mat == nil {
		return sc.defaultMaterialGroup
	}
	key := mat.baseColorTex + "|" + mat.normalTex + "|" + mat.mrTex + "|" + mat.emissiveTex
	if bg, ok := sc.materialGroups[key]; ok {
		return bg
	}
	bg := sc.makeMaterialBindGroup(
		sc.getTexture(resolver, mat.baseColorTex, true, sc.fallbackWhite),
		sc.getTexture(resolver, mat.normalTex, false, sc.fallbackNormal),
		sc.getTexture(resolver, mat.mrTex, false, sc.fallbackWhite),
		sc.getTexture(resolver, mat.emissiveTex, true, sc.fallbackBlack),
	)
	if bg == nil {
		return sc.defaultMaterialGroup
	}
	sc.materialGroups[key] = bg
	return bg
}

// buildVertexDataSkinned создаёт vertex buffer для скиннинга: pos(12)+normal(12)+uv(8)+joints(16)+weights(16)=64 bytes.
func buildVertexDataSkinned(positions, normals, uvs []float32, joints []uint16, weights []float32, indices []uint32) []byte {
	if len(positions) == 0 || len(indices) == 0 {
		return nil
	}
	vertexCount := len(indices)
	stride := 64
	data := make([]byte, vertexCount*stride)
	for i := 0; i < vertexCount; i++ {
		idx := int(indices[i])
		off := i * stride
		if idx*3+2 < len(positions) {
			binary.LittleEndian.PutUint32(data[off:], math.Float32bits(positions[idx*3]))
			binary.LittleEndian.PutUint32(data[off+4:], math.Float32bits(positions[idx*3+1]))
			binary.LittleEndian.PutUint32(data[off+8:], math.Float32bits(positions[idx*3+2]))
		}
		if len(normals) > 0 && idx*3+2 < len(normals) {
			binary.LittleEndian.PutUint32(data[off+12:], math.Float32bits(normals[idx*3]))
			binary.LittleEndian.PutUint32(data[off+16:], math.Float32bits(normals[idx*3+1]))
			binary.LittleEndian.PutUint32(data[off+20:], math.Float32bits(normals[idx*3+2]))
		} else {
			binary.LittleEndian.PutUint32(data[off+12:], math.Float32bits(0))
			binary.LittleEndian.PutUint32(data[off+16:], math.Float32bits(1))
			binary.LittleEndian.PutUint32(data[off+20:], math.Float32bits(0))
		}
		if len(uvs) > 0 && idx*2+1 < len(uvs) {
			binary.LittleEndian.PutUint32(data[off+24:], math.Float32bits(uvs[idx*2]))
			binary.LittleEndian.PutUint32(data[off+28:], math.Float32bits(uvs[idx*2+1]))
		}
		if len(joints) >= idx*4+4 && len(weights) >= idx*4+4 {
			for j := 0; j < 4; j++ {
				binary.LittleEndian.PutUint32(data[off+32+j*4:], math.Float32bits(float32(joints[idx*4+j])))
			}
			for j := 0; j < 4; j++ {
				binary.LittleEndian.PutUint32(data[off+48+j*4:], math.Float32bits(weights[idx*4+j]))
			}
		} else {
			binary.LittleEndian.PutUint32(data[off+32:], math.Float32bits(0))
			binary.LittleEndian.PutUint32(data[off+36:], math.Float32bits(0))
			binary.LittleEndian.PutUint32(data[off+40:], math.Float32bits(0))
			binary.LittleEndian.PutUint32(data[off+44:], math.Float32bits(0))
			binary.LittleEndian.PutUint32(data[off+48:], math.Float32bits(1))
			binary.LittleEndian.PutUint32(data[off+52:], math.Float32bits(0))
			binary.LittleEndian.PutUint32(data[off+56:], math.Float32bits(0))
			binary.LittleEndian.PutUint32(data[off+60:], math.Float32bits(0))
		}
	}
	return data
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
		// LOD-меши идентифицируются путём (Mesh.LODRefs), а не assetID.
		if mesh, err := resolver.ResolveMeshByPath(meshAssetID); err == nil {
			return mesh.Positions, mesh.Normals, mesh.UV0, mesh.Indices
		}
	}
	cube := render.CreateCube()
	return cube.Vertices, cube.Normals, cube.UVs, cube.Indices
}

// maxDrawDistance — сущности дальше этого радиуса не рисуются (масштабируемость больших сцен).
const maxDrawDistance = 400

// selectLODMesh выбирает LOD-меш по дистанции до камеры (пороги как в renderer3d:
// 0–12 = LOD0, 12–35 = LOD1, 35+ = LOD2). Возвращает путь LOD-меша или исходный
// assetID, если LOD нет. Применяется только к static-мешам.
func selectLODMesh(resolver *asset.Resolver, meshAssetID string, dist float32) string {
	if resolver == nil || meshAssetID == "" || dist <= 12 {
		return meshAssetID
	}
	mesh, err := resolver.ResolveMeshByAssetID(meshAssetID)
	if err != nil || mesh == nil {
		return meshAssetID
	}
	if len(mesh.LODRefs) >= 2 && dist > 35 {
		if lod2, err := resolver.ResolveMeshByPath(mesh.LODRefs[1]); err == nil && lod2 != nil {
			return mesh.LODRefs[1]
		}
	} else if len(mesh.LODRefs) >= 1 {
		if lod1, err := resolver.ResolveMeshByPath(mesh.LODRefs[0]); err == nil && lod1 != nil {
			return mesh.LODRefs[0]
		}
	}
	return meshAssetID
}

// meshDataWithSkin загружает меш и данные скина (если есть).
func meshDataWithSkin(resolver *asset.Resolver, meshAssetID string) (positions, normals, uvs []float32, joints []uint16, weights []float32, indices []uint32) {
	if resolver != nil && meshAssetID != "" {
		if mesh, err := resolver.ResolveMeshByAssetID(meshAssetID); err == nil {
			return mesh.Positions, mesh.Normals, mesh.UV0, mesh.Joints, mesh.Weights, mesh.Indices
		}
	}
	cube := render.CreateCube()
	return cube.Vertices, cube.Normals, cube.UVs, nil, nil, cube.Indices
}

// cachedMesh — закэшированный vertex buffer для mesh (переиспользуется между кадрами)
type cachedMesh struct {
	vertexBuf   *wgpu.Buffer
	vertexCount uint32
}

// sceneState holds GPU resources for rendering a 3D scene.
// stencilDisabled — валидное «стенсил выключен» состояние: wgpu v0.23 паникует
// на нулевом CompareFunction (Undefined) в DepthStencilState (conv.rs).
var stencilDisabled = wgpu.StencilFaceState{
	Compare:     wgpu.CompareFunctionAlways,
	FailOp:      wgpu.StencilOperationKeep,
	DepthFailOp: wgpu.StencilOperationKeep,
	PassOp:      wgpu.StencilOperationKeep,
}

type sceneState struct {
	device          *wgpu.Device
	queue           *wgpu.Queue
	pipeline        *wgpu.RenderPipeline
	uniformBuffer   *wgpu.Buffer
	bindGroup       *wgpu.BindGroup
	bindGroupShadow *wgpu.BindGroup
	cubeVertexBuf   *wgpu.Buffer
	cubeVertexCount uint32

	meshCache map[string]*cachedMesh // meshAssetID -> vertex buffer (для 10k+ tris без пересоздания)

	shadowMap       *wgpu.Texture
	shadowView      *wgpu.TextureView
	shadowPipeline  *wgpu.RenderPipeline
	shadowUniform   *wgpu.Buffer
	shadowBindGroup *wgpu.BindGroup
	shadowSampler   *wgpu.Sampler

	// Point light shadows: depth-array из 6 граней (cubemap), линейная глубина через frag_depth.
	pointShadowMap       *wgpu.Texture
	pointShadowView      *wgpu.TextureView // view для семплирования (2d-array)
	pointShadowFaces     [6]*wgpu.TextureView
	pointShadowPipeline  *wgpu.RenderPipeline
	pointShadowUniform   *wgpu.Buffer // 144 bytes: view_proj + model + light_pos + light_far
	pointShadowBindGroup *wgpu.BindGroup

	// Spot light shadow: перспективная 2D-карта (как directional, но perspective).
	spotShadowMap  *wgpu.Texture
	spotShadowView *wgpu.TextureView

	// Point light shadows для skinned-мешей (linear depth + skinning в vertex).
	pointShadowSkinnedPipeline  *wgpu.RenderPipeline
	pointShadowSkinnedUniform   *wgpu.Buffer // 144 bytes
	pointShadowSkinnedBindGroup *wgpu.BindGroup

	// Skeletal skinning
	skinnedPipeline        *wgpu.RenderPipeline
	skinnedUniform         *wgpu.Buffer // PBR uniforms + model at 64
	boneUniform            *wgpu.Buffer
	skinnedBindGroup       *wgpu.BindGroup
	skinnedBindGroupShadow *wgpu.BindGroup
	meshSkinnedCache       map[string]*cachedMesh // meshAssetID (skinned) -> vertex buffer
	shadowSkinnedPipeline  *wgpu.RenderPipeline   // shadow pass с реальным скиннингом
	shadowSkinnedBindGroup *wgpu.BindGroup        // shadow uniforms + bones
	shadowSkinnedUniform   *wgpu.Buffer           // 128 bytes: light_view_proj + model

	// Текстуры материалов (группа 2): кэш текстур и material bind groups, fallback-текстуры
	textureCache   map[string]*wgpu.Texture   // путь .texture.json -> GPU texture
	materialGroups map[string]*wgpu.BindGroup // ключ (пути текстур) -> bind group
	fallbackWhite  *wgpu.Texture              // 1x1 белая (color/MR по умолчанию)
	fallbackNormal *wgpu.Texture              // 1x1 (128,128,255) — flat normal
	fallbackBlack  *wgpu.Texture              // 1x1 чёрная (emissive по умолчанию)
	textureSampler *wgpu.Sampler

	// IBL (группа 3): процедурные env/irradiance кубомапы.
	envTex       *wgpu.Texture
	envView      *wgpu.TextureView
	irrTex       *wgpu.Texture
	irrView      *wgpu.TextureView
	envSampler   *wgpu.Sampler
	envBindGroup *wgpu.BindGroup

	// Пост-процесс: HDR offscreen (16F) + bloom (полуразрешение) + composite в surface.
	sceneTex          *wgpu.Texture
	sceneView         *wgpu.TextureView
	bloomTexA         *wgpu.Texture
	bloomViewA        *wgpu.TextureView
	bloomTexB         *wgpu.Texture
	bloomViewB        *wgpu.TextureView
	brightPipeline    *wgpu.RenderPipeline
	blurPipeline      *wgpu.RenderPipeline
	compositePipeline *wgpu.RenderPipeline
	postUniform       *wgpu.Buffer          // 32 bytes: direction vec2, intensity, vignette, dof*
	postBGLBright     *wgpu.BindGroupLayout // биндинги 0,1
	postBGLBlur       *wgpu.BindGroupLayout // 0,1 + юниформ (3)
	postBGLComposite  *wgpu.BindGroupLayout // 0..3 (сцена + bloom + юниформ)
	brightBG          *wgpu.BindGroup
	blurAB, blurBA    *wgpu.BindGroup
	compositeBG       *wgpu.BindGroup
	postW, postH      int

	// HUD-оверлей (полосы/текст поверх кадра), ленивая инициализация.
	hud *hudState

	defaultMaterialGroup *wgpu.BindGroup // материал без текстур (fallback cube и т.п.)

	// MSAA: multisample color + depth, резолв в surface view в конце main pass.
	msaaColor     *wgpu.Texture
	msaaView      *wgpu.TextureView
	msaaDepth     *wgpu.Texture
	msaaDepthView *wgpu.TextureView
	msaaW, msaaH  int
	msaaSamples   uint32
}

// ensureMSAA создаёт (или пересоздаёт при смене размера) multisample color/depth
// текстуры для main pass. Формат color совпадает с форматом surface.
func (sc *sceneState) ensureMSAA(device *wgpu.Device, format wgpu.TextureFormat, width, height int, samples uint32) error {
	if sc.msaaColor != nil && sc.msaaW == width && sc.msaaH == height && sc.msaaSamples == samples {
		return nil
	}
	sc.releaseMSAA()
	sc.msaaSamples = samples
	if width <= 0 || height <= 0 {
		return nil
	}
	color, err := device.CreateTexture(&wgpu.TextureDescriptor{
		Label:         "msaa color",
		Size:          wgpu.Extent3D{Width: uint32(width), Height: uint32(height), DepthOrArrayLayers: 1},
		MipLevelCount: 1,
		SampleCount:   samples,
		Dimension:     wgpu.TextureDimension2D,
		Format:        format,
		Usage:         wgpu.TextureUsageRenderAttachment,
	})
	if err != nil {
		return err
	}
	depth, err := device.CreateTexture(&wgpu.TextureDescriptor{
		Label:         "msaa depth",
		Size:          wgpu.Extent3D{Width: uint32(width), Height: uint32(height), DepthOrArrayLayers: 1},
		MipLevelCount: 1,
		SampleCount:   samples,
		Dimension:     wgpu.TextureDimension2D,
		Format:        wgpu.TextureFormatDepth32Float,
		Usage:         wgpu.TextureUsageRenderAttachment,
	})
	if err != nil {
		color.Release()
		return err
	}
	colorView, err := color.CreateView(nil)
	if err != nil {
		color.Release()
		depth.Release()
		return err
	}
	depthView, err := depth.CreateView(nil)
	if err != nil {
		colorView.Release()
		color.Release()
		depth.Release()
		return err
	}
	sc.msaaColor = color
	sc.msaaDepth = depth
	sc.msaaView = colorView
	sc.msaaDepthView = depthView
	sc.msaaW = width
	sc.msaaH = height
	return nil
}

// releaseMSAA освобождает multisample ресурсы main pass.
func (sc *sceneState) releaseMSAA() {
	if sc.msaaView != nil {
		sc.msaaView.Release()
		sc.msaaView = nil
	}
	if sc.msaaColor != nil {
		sc.msaaColor.Release()
		sc.msaaColor = nil
	}
	if sc.msaaDepthView != nil {
		sc.msaaDepthView.Release()
		sc.msaaDepthView = nil
	}
	if sc.msaaDepth != nil {
		sc.msaaDepth.Release()
		sc.msaaDepth = nil
	}
	sc.msaaW, sc.msaaH = 0, 0
	sc.msaaSamples = 0
}

// ensurePostTargets создаёт/пересоздаёт HDR-таргет сцены и bloom-текстуры
// (полуразрешение) при первом кадре или смене размера окна.
func (sc *sceneState) ensurePostTargets(device *wgpu.Device, width, height int) error {
	if sc.sceneTex != nil && sc.postW == width && sc.postH == height {
		return nil
	}
	sc.releasePost()
	if width <= 0 || height <= 0 {
		return nil
	}
	bw, bh := width/2, height/2
	if bw < 1 {
		bw = 1
	}
	if bh < 1 {
		bh = 1
	}
	create := func(label string, w, h int, usage wgpu.TextureUsage) (*wgpu.Texture, *wgpu.TextureView, error) {
		tex, err := device.CreateTexture(&wgpu.TextureDescriptor{
			Label:         label,
			Size:          wgpu.Extent3D{Width: uint32(w), Height: uint32(h), DepthOrArrayLayers: 1},
			MipLevelCount: 1,
			SampleCount:   1,
			Dimension:     wgpu.TextureDimension2D,
			Format:        wgpu.TextureFormatRGBA16Float,
			Usage:         usage,
		})
		if err != nil {
			return nil, nil, err
		}
		v, err := tex.CreateView(nil)
		if err != nil {
			tex.Release()
			return nil, nil, err
		}
		return tex, v, nil
	}
	tex, v, err := create("post scene", width, height, wgpu.TextureUsageRenderAttachment|wgpu.TextureUsageTextureBinding)
	if err != nil {
		return err
	}
	sc.sceneTex, sc.sceneView = tex, v
	ta, va, err := create("bloom a", bw, bh, wgpu.TextureUsageRenderAttachment|wgpu.TextureUsageTextureBinding)
	if err != nil {
		sc.releasePost()
		return err
	}
	sc.bloomTexA, sc.bloomViewA = ta, va
	tb, vb, err := create("bloom b", bw, bh, wgpu.TextureUsageRenderAttachment|wgpu.TextureUsageTextureBinding)
	if err != nil {
		sc.releasePost()
		return err
	}
	sc.bloomTexB, sc.bloomViewB = tb, vb
	sc.postW, sc.postH = width, height

	// Bind groups пост-процесса: у каждого пайплайна свой layout — bright
	// семплирует только сцену, blur добавляет юниформы, composite — всё + bloom.
	mk := func(bgl *wgpu.BindGroupLayout, entries []wgpu.BindGroupEntry) *wgpu.BindGroup {
		if bgl == nil {
			return nil
		}
		bg, err := device.CreateBindGroup(&wgpu.BindGroupDescriptor{Layout: bgl, Entries: entries})
		if err != nil {
			return nil
		}
		return bg
	}
	for _, bg := range []**wgpu.BindGroup{&sc.brightBG, &sc.blurAB, &sc.blurBA, &sc.compositeBG} {
		if *bg != nil {
			(*bg).Release()
			*bg = nil
		}
	}
	sc.brightBG = mk(sc.postBGLBright, []wgpu.BindGroupEntry{
		{Binding: 0, TextureView: sc.sceneView},
		{Binding: 1, Sampler: sc.envSampler},
	})
	sc.blurAB = mk(sc.postBGLBlur, []wgpu.BindGroupEntry{
		{Binding: 0, TextureView: sc.bloomViewA},
		{Binding: 1, Sampler: sc.envSampler},
		{Binding: 3, Buffer: sc.postUniform, Size: 32},
	})
	sc.blurBA = mk(sc.postBGLBlur, []wgpu.BindGroupEntry{
		{Binding: 0, TextureView: sc.bloomViewB},
		{Binding: 1, Sampler: sc.envSampler},
		{Binding: 3, Buffer: sc.postUniform, Size: 32},
	})
	sc.compositeBG = mk(sc.postBGLComposite, []wgpu.BindGroupEntry{
		{Binding: 0, TextureView: sc.sceneView},
		{Binding: 1, Sampler: sc.envSampler},
		{Binding: 2, TextureView: sc.bloomViewA},
		{Binding: 3, Buffer: sc.postUniform, Size: 32},
	})
	return nil
}

// releasePost освобождает ресурсы пост-процесса (текстуры пересоздаются
// при необходимости; пайплайны/биндинги — только в Destroy).
func (sc *sceneState) releasePost() {
	for _, v := range []**wgpu.TextureView{&sc.sceneView, &sc.bloomViewA, &sc.bloomViewB} {
		if *v != nil {
			(*v).Release()
			*v = nil
		}
	}
	for _, t := range []**wgpu.Texture{&sc.sceneTex, &sc.bloomTexA, &sc.bloomTexB} {
		if *t != nil {
			(*t).Release()
			*t = nil
		}
	}
	sc.postW, sc.postH = 0, 0
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
			CullMode:         wgpu.CullModeNone, // меши репо CW, процедурный куб CCW — рисуем оба направления
		},
		DepthStencil: &wgpu.DepthStencilState{
			Format:            wgpu.TextureFormatDepth32Float,
			DepthWriteEnabled: true,
			DepthCompare:      wgpu.CompareFunctionLessEqual,
			StencilFront:      stencilDisabled,
			StencilBack:       stencilDisabled,
			StencilReadMask:   0xFFFFFFFF,
			StencilWriteMask:  0xFFFFFFFF,
		},
		Multisample: wgpu.MultisampleState{Count: 4, Mask: 0xFFFFFFFF},
		Fragment: &wgpu.FragmentState{
			Module:     shader,
			EntryPoint: "fs_main",
			Targets: []wgpu.ColorTargetState{
				// Alpha blending: OPAQUE материалы имеют alpha=1 (без видимого эффекта),
				// BLEND-материалы получают честную прозрачность.
				{Format: wgpu.TextureFormatRGBA16Float, Blend: &wgpu.BlendStateAlphaBlending, WriteMask: wgpu.ColorWriteMaskAll},
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
		Label:         "shadow map",
		Size:          wgpu.Extent3D{Width: 2048, Height: 2048, DepthOrArrayLayers: 1},
		MipLevelCount: 1,
		SampleCount:   1,
		Dimension:     wgpu.TextureDimension2D,
		Format:        wgpu.TextureFormatDepth32Float,
		Usage:         wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageTextureBinding,
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
		Compare:       wgpu.CompareFunctionLessEqual,
		AddressModeU:  wgpu.AddressModeClampToEdge,
		AddressModeV:  wgpu.AddressModeClampToEdge,
		MaxAnisotropy: 1,
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
			Topology:  wgpu.PrimitiveTopologyTriangleList,
			FrontFace: wgpu.FrontFaceCCW,
			CullMode:  wgpu.CullModeNone, // см. комментарий выше: смешанные намотки
		},
		DepthStencil: &wgpu.DepthStencilState{
			Format:            wgpu.TextureFormatDepth32Float,
			DepthWriteEnabled: true,
			DepthCompare:      wgpu.CompareFunctionLessEqual,
			StencilFront:      stencilDisabled,
			StencilBack:       stencilDisabled,
			StencilReadMask:   0xFFFFFFFF,
			StencilWriteMask:  0xFFFFFFFF,
		},
		Multisample: wgpu.MultisampleState{Count: 1, Mask: 0xFFFFFFFF},
		Fragment:    nil,
	})
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
	// Skinned shadow pass: отдельный шейдер, который скинит вершины (bones в vertex shader).
	shadowSkinnedShader, err := s.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:          "shadow skinned shader",
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{Code: shadowSkinnedWGSL},
	})
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
	shadowSkinnedUb, err := s.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "shadow skinned uniforms",
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
		Size:  128,
	})
	if err != nil {
		shadowSkinnedShader.Release()
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
	// Bone matrices (общий для main skinned pass и skinned shadow pass)
	boneUb, err := s.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "bone matrices",
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
		Size:  boneUniformsSize,
	})
	if err != nil {
		shadowSkinnedShader.Release()
		shadowSkinnedUb.Release()
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
	shadowSkinnedPl, err := s.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "Shadow Skinned Pipeline",
		Vertex: wgpu.VertexState{
			Module:     shadowSkinnedShader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{
				{ArrayStride: 64, StepMode: wgpu.VertexStepModeVertex,
					Attributes: []wgpu.VertexAttribute{
						{Format: wgpu.VertexFormatFloat32x3, Offset: 0, ShaderLocation: 0},
						{Format: wgpu.VertexFormatFloat32x3, Offset: 12, ShaderLocation: 1},
						{Format: wgpu.VertexFormatFloat32x2, Offset: 24, ShaderLocation: 2},
						{Format: wgpu.VertexFormatFloat32x4, Offset: 32, ShaderLocation: 3},
						{Format: wgpu.VertexFormatFloat32x4, Offset: 48, ShaderLocation: 4},
					},
				},
			},
		},
		Primitive: wgpu.PrimitiveState{
			Topology:  wgpu.PrimitiveTopologyTriangleList,
			FrontFace: wgpu.FrontFaceCCW,
			CullMode:  wgpu.CullModeNone, // см. комментарий выше: смешанные намотки
		},
		DepthStencil: &wgpu.DepthStencilState{
			Format:            wgpu.TextureFormatDepth32Float,
			DepthWriteEnabled: true,
			DepthCompare:      wgpu.CompareFunctionLessEqual,
			StencilFront:      stencilDisabled,
			StencilBack:       stencilDisabled,
			StencilReadMask:   0xFFFFFFFF,
			StencilWriteMask:  0xFFFFFFFF,
		},
		Multisample: wgpu.MultisampleState{Count: 1, Mask: 0xFFFFFFFF},
		Fragment:    nil,
	})
	if err != nil {
		shadowSkinnedShader.Release()
		shadowSkinnedUb.Release()
		boneUb.Release()
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
	shadowSkinnedBgl := shadowSkinnedPl.GetBindGroupLayout(0)
	shadowSkinnedBg, err := s.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout: shadowSkinnedBgl,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: shadowSkinnedUb, Size: 128},
			{Binding: 1, Buffer: boneUb, Size: boneUniformsSize},
		},
	})
	shadowSkinnedBgl.Release()
	if err != nil {
		shadowSkinnedPl.Release()
		shadowSkinnedShader.Release()
		shadowSkinnedUb.Release()
		boneUb.Release()
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
	shadowSkinnedShader.Release()

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

	// --- Point light shadows: depth-array кубомапа (6 граней), линейная глубина через frag_depth ---
	pointShadowTex, err := s.device.CreateTexture(&wgpu.TextureDescriptor{
		Label:         "point shadow map",
		Size:          wgpu.Extent3D{Width: 512, Height: 512, DepthOrArrayLayers: 6},
		MipLevelCount: 1,
		SampleCount:   1,
		Dimension:     wgpu.TextureDimension2D,
		Format:        wgpu.TextureFormatDepth32Float,
		Usage:         wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageTextureBinding,
	})
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
	pointShadowView, err := pointShadowTex.CreateView(&wgpu.TextureViewDescriptor{
		Label:           "point shadow array view",
		Dimension:       wgpu.TextureViewDimension2DArray,
		ArrayLayerCount: 6,
		MipLevelCount:   1,
	})
	if err != nil {
		pointShadowTex.Release()
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
	var pointShadowFaces [6]*wgpu.TextureView
	for i := 0; i < 6; i++ {
		fv, err := pointShadowTex.CreateView(&wgpu.TextureViewDescriptor{
			Label:           fmt.Sprintf("point shadow face %d", i),
			Dimension:       wgpu.TextureViewDimension2D,
			BaseArrayLayer:  uint32(i),
			ArrayLayerCount: 1,
			MipLevelCount:   1,
		})
		if err != nil {
			for j := 0; j < i; j++ {
				pointShadowFaces[j].Release()
			}
			pointShadowView.Release()
			pointShadowTex.Release()
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
		pointShadowFaces[i] = fv
	}
	pointShadowShader, err := s.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:          "point shadow shader",
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{Code: pointShadowWGSL},
	})
	if err != nil {
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
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
	pointShadowUb, err := s.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "point shadow uniforms",
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
		Size:  144,
	})
	if err != nil {
		pointShadowShader.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
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
	pointShadowPl, err := s.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "Point Shadow Pipeline",
		Vertex: wgpu.VertexState{
			Module:     pointShadowShader,
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
			Topology:  wgpu.PrimitiveTopologyTriangleList,
			FrontFace: wgpu.FrontFaceCCW,
			CullMode:  wgpu.CullModeNone, // см. комментарий выше: смешанные намотки
		},
		DepthStencil: &wgpu.DepthStencilState{
			Format:            wgpu.TextureFormatDepth32Float,
			DepthWriteEnabled: true,
			DepthCompare:      wgpu.CompareFunctionLessEqual,
			StencilFront:      stencilDisabled,
			StencilBack:       stencilDisabled,
			StencilReadMask:   0xFFFFFFFF,
			StencilWriteMask:  0xFFFFFFFF,
		},
		Multisample: wgpu.MultisampleState{Count: 1, Mask: 0xFFFFFFFF},
		Fragment: &wgpu.FragmentState{
			Module:     pointShadowShader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{},
		},
	})
	pointShadowShader.Release()
	if err != nil {
		pointShadowUb.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
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
	pointShadowBgl := pointShadowPl.GetBindGroupLayout(0)
	pointShadowBg, err := s.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout: pointShadowBgl,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: pointShadowUb, Size: 144},
		},
	})
	pointShadowBgl.Release()
	if err != nil {
		pointShadowPl.Release()
		pointShadowUb.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
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

	// Point light shadows для skinned-мешей.
	pointShadowSkinnedShader, err := s.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:          "point shadow skinned shader",
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{Code: pointShadowSkinnedWGSL},
	})
	if err != nil {
		pointShadowBg.Release()
		pointShadowPl.Release()
		pointShadowUb.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
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
	pointShadowSkinnedUb, err := s.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "point shadow skinned uniforms",
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
		Size:  144,
	})
	if err != nil {
		pointShadowSkinnedShader.Release()
		pointShadowBg.Release()
		pointShadowPl.Release()
		pointShadowUb.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
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
	pointShadowSkinnedPl, err := s.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "Point Shadow Skinned Pipeline",
		Vertex: wgpu.VertexState{
			Module:     pointShadowSkinnedShader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{
				{ArrayStride: 64, StepMode: wgpu.VertexStepModeVertex,
					Attributes: []wgpu.VertexAttribute{
						{Format: wgpu.VertexFormatFloat32x3, Offset: 0, ShaderLocation: 0},
						{Format: wgpu.VertexFormatFloat32x3, Offset: 12, ShaderLocation: 1},
						{Format: wgpu.VertexFormatFloat32x2, Offset: 24, ShaderLocation: 2},
						{Format: wgpu.VertexFormatFloat32x4, Offset: 32, ShaderLocation: 3},
						{Format: wgpu.VertexFormatFloat32x4, Offset: 48, ShaderLocation: 4},
					},
				},
			},
		},
		Primitive: wgpu.PrimitiveState{
			Topology:  wgpu.PrimitiveTopologyTriangleList,
			FrontFace: wgpu.FrontFaceCCW,
			CullMode:  wgpu.CullModeNone, // см. комментарий выше: смешанные намотки
		},
		DepthStencil: &wgpu.DepthStencilState{
			Format:            wgpu.TextureFormatDepth32Float,
			DepthWriteEnabled: true,
			DepthCompare:      wgpu.CompareFunctionLessEqual,
			StencilFront:      stencilDisabled,
			StencilBack:       stencilDisabled,
			StencilReadMask:   0xFFFFFFFF,
			StencilWriteMask:  0xFFFFFFFF,
		},
		Multisample: wgpu.MultisampleState{Count: 1, Mask: 0xFFFFFFFF},
		Fragment: &wgpu.FragmentState{
			Module:     pointShadowSkinnedShader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{},
		},
	})
	pointShadowSkinnedShader.Release()
	if err != nil {
		pointShadowSkinnedUb.Release()
		pointShadowBg.Release()
		pointShadowPl.Release()
		pointShadowUb.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
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
	psSkinnedBgl := pointShadowSkinnedPl.GetBindGroupLayout(0)
	pointShadowSkinnedBg, err := s.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout: psSkinnedBgl,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: pointShadowSkinnedUb, Size: 144},
			{Binding: 1, Buffer: boneUb, Size: boneUniformsSize},
		},
	})
	psSkinnedBgl.Release()
	if err != nil {
		pointShadowSkinnedPl.Release()
		pointShadowSkinnedUb.Release()
		pointShadowBg.Release()
		pointShadowPl.Release()
		pointShadowUb.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
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

	// Spot light shadow map: 1024x1024 перспективная карта прожектора.
	spotShadowTex, err := s.device.CreateTexture(&wgpu.TextureDescriptor{
		Label:         "spot shadow map",
		Size:          wgpu.Extent3D{Width: 1024, Height: 1024, DepthOrArrayLayers: 1},
		MipLevelCount: 1,
		SampleCount:   1,
		Dimension:     wgpu.TextureDimension2D,
		Format:        wgpu.TextureFormatDepth32Float,
		Usage:         wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageTextureBinding,
	})
	if err != nil {
		pointShadowBg.Release()
		pointShadowPl.Release()
		pointShadowUb.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
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
	spotShadowView, err := spotShadowTex.CreateView(nil)
	if err != nil {
		spotShadowTex.Release()
		pointShadowBg.Release()
		pointShadowPl.Release()
		pointShadowUb.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
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
			{Binding: 2, TextureView: pointShadowView},
			{Binding: 3, Sampler: shadowSampler},
			{Binding: 4, TextureView: spotShadowView},
			{Binding: 5, Sampler: shadowSampler},
		},
	})
	bgl1.Release()
	if err != nil {
		pointShadowBg.Release()
		pointShadowPl.Release()
		pointShadowUb.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
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

	// Skinned pipeline
	skinnedShader, err := s.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:          "skinned shader",
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{Code: shaderSkinnedWGSL},
	})
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
	skinnedUb, err := s.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "skinned uniforms",
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
		Size:  pbrUniformsSize,
	})
	if err != nil {
		skinnedShader.Release()
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
	skinnedPl, err := s.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "Skinned Pipeline",
		Vertex: wgpu.VertexState{
			Module:     skinnedShader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{
				{
					ArrayStride: 64,
					StepMode:    wgpu.VertexStepModeVertex,
					Attributes: []wgpu.VertexAttribute{
						{Format: wgpu.VertexFormatFloat32x3, Offset: 0, ShaderLocation: 0},
						{Format: wgpu.VertexFormatFloat32x3, Offset: 12, ShaderLocation: 1},
						{Format: wgpu.VertexFormatFloat32x2, Offset: 24, ShaderLocation: 2},
						{Format: wgpu.VertexFormatFloat32x4, Offset: 32, ShaderLocation: 3},
						{Format: wgpu.VertexFormatFloat32x4, Offset: 48, ShaderLocation: 4},
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
		DepthStencil: &wgpu.DepthStencilState{
			Format:            wgpu.TextureFormatDepth32Float,
			DepthWriteEnabled: true,
			DepthCompare:      wgpu.CompareFunctionLessEqual,
			StencilFront:      stencilDisabled,
			StencilBack:       stencilDisabled,
			StencilReadMask:   0xFFFFFFFF,
			StencilWriteMask:  0xFFFFFFFF,
		},
		Multisample: wgpu.MultisampleState{Count: 4, Mask: 0xFFFFFFFF},
		Fragment: &wgpu.FragmentState{
			Module:     skinnedShader,
			EntryPoint: "fs_main",
			Targets: []wgpu.ColorTargetState{
				{Format: wgpu.TextureFormatRGBA16Float, Blend: &wgpu.BlendStateAlphaBlending, WriteMask: wgpu.ColorWriteMaskAll},
			},
		},
	})
	skinnedShader.Release()
	if err != nil {
		boneUb.Release()
		skinnedUb.Release()
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
	skinnedBgl := skinnedPl.GetBindGroupLayout(0)
	skinnedBg, err := s.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout: skinnedBgl,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: skinnedUb, Size: pbrUniformsSize},
			{Binding: 1, Buffer: boneUb, Size: boneUniformsSize},
		},
	})
	skinnedBgl.Release()
	if err != nil {
		skinnedPl.Release()
		boneUb.Release()
		skinnedUb.Release()
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
	skinnedBgl1 := skinnedPl.GetBindGroupLayout(1)
	skinnedBgShadow, err := s.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout: skinnedBgl1,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, TextureView: shadowView},
			{Binding: 1, Sampler: shadowSampler},
			{Binding: 2, TextureView: pointShadowView},
			{Binding: 3, Sampler: shadowSampler},
			{Binding: 4, TextureView: spotShadowView},
			{Binding: 5, Sampler: shadowSampler},
		},
	})
	skinnedBgl1.Release()
	if err != nil {
		pointShadowBg.Release()
		pointShadowPl.Release()
		pointShadowUb.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
		skinnedBg.Release()
		skinnedPl.Release()
		boneUb.Release()
		skinnedUb.Release()
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

	// IBL: процедурные env/irradiance кубомапы (группа 3) + свой sampler.
	envTexR, envViewR, irrTexR, irrViewR, err := createEnvTextures(s.device, s.queue)
	if err != nil {
		skinnedBgShadow.Release()
		skinnedBg.Release()
		skinnedPl.Release()
		boneUb.Release()
		skinnedUb.Release()
		shadowSkinnedUb.Release()
		shadowPl.Release()
		shadowUb.Release()
		shadowSampler.Release()
		shadowView.Release()
		shadowTex.Release()
		pointShadowBg.Release()
		pointShadowPl.Release()
		pointShadowUb.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
		pipeline.Release()
		cubeBuf.Release()
		ub.Release()
		bg.Release()
		return err
	}
	envSmp, err := s.device.CreateSampler(&wgpu.SamplerDescriptor{
		Label:         "env sampler",
		AddressModeU:  wgpu.AddressModeClampToEdge,
		AddressModeV:  wgpu.AddressModeClampToEdge,
		AddressModeW:  wgpu.AddressModeClampToEdge,
		MagFilter:     wgpu.FilterModeLinear,
		MinFilter:     wgpu.FilterModeLinear,
		MipmapFilter:  wgpu.MipmapFilterModeLinear,
		MaxAnisotropy: 1,
	})
	if err != nil {
		irrViewR.Release()
		irrTexR.Release()
		envViewR.Release()
		envTexR.Release()
		skinnedBgShadow.Release()
		skinnedBg.Release()
		skinnedPl.Release()
		boneUb.Release()
		skinnedUb.Release()
		shadowSkinnedUb.Release()
		shadowPl.Release()
		shadowUb.Release()
		shadowSampler.Release()
		shadowView.Release()
		shadowTex.Release()
		pointShadowBg.Release()
		pointShadowPl.Release()
		pointShadowUb.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
		pipeline.Release()
		cubeBuf.Release()
		ub.Release()
		bg.Release()
		return err
	}
	envBgl := pipeline.GetBindGroupLayout(3) // группа 3 (env/IBL) мешевого пайплайна — та же группа ставится в обоих проходах
	envBg, err := s.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout: envBgl,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, TextureView: envViewR},
			{Binding: 1, Sampler: envSmp},
			{Binding: 2, TextureView: irrViewR},
			{Binding: 3, Sampler: envSmp},
		},
	})
	envBgl.Release()
	if err != nil {
		envSmp.Release()
		irrViewR.Release()
		irrTexR.Release()
		envViewR.Release()
		envTexR.Release()
		skinnedBgShadow.Release()
		skinnedBg.Release()
		skinnedPl.Release()
		boneUb.Release()
		skinnedUb.Release()
		shadowSkinnedUb.Release()
		shadowPl.Release()
		shadowUb.Release()
		shadowSampler.Release()
		shadowView.Release()
		shadowTex.Release()
		pointShadowBg.Release()
		pointShadowPl.Release()
		pointShadowUb.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
		pipeline.Release()
		cubeBuf.Release()
		ub.Release()
		bg.Release()
		return err
	}

	// --- Пост-процесс: bright / blur / composite ---
	releaseUpToEnv := func() {
		envBg.Release()
		envSmp.Release()
		irrViewR.Release()
		irrTexR.Release()
		envViewR.Release()
		envTexR.Release()
		skinnedBgShadow.Release()
		skinnedBg.Release()
		skinnedPl.Release()
		boneUb.Release()
		skinnedUb.Release()
		shadowSkinnedUb.Release()
		shadowSkinnedBg.Release()
		shadowSkinnedPl.Release()
		shadowPl.Release()
		shadowUb.Release()
		shadowSampler.Release()
		shadowView.Release()
		shadowTex.Release()
		pointShadowBg.Release()
		pointShadowPl.Release()
		pointShadowUb.Release()
		for i := 0; i < 6; i++ {
			pointShadowFaces[i].Release()
		}
		pointShadowView.Release()
		pointShadowTex.Release()
		pipeline.Release()
		cubeBuf.Release()
		ub.Release()
		bg.Release()
	}
	postShader, err := s.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:          "post shader",
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{Code: postprocessWGSL},
	})
	if err != nil {
		releaseUpToEnv()
		return err
	}
	postUb, err := s.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "post uniforms",
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
		Size:  32,
	})
	if err != nil {
		postShader.Release()
		releaseUpToEnv()
		return err
	}

	// Явный layout пост-процесса: auto-layout bright-пайплайна содержит только
	// его 2 биндинга, а composite требует все 4 (2 текстуры + сэмплер + юниформ).
	postBGLBright, err := s.device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "post bright BGL",
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageFragment, Texture: wgpu.TextureBindingLayout{SampleType: wgpu.TextureSampleTypeFloat, ViewDimension: wgpu.TextureViewDimension2D}},
			{Binding: 1, Visibility: wgpu.ShaderStageFragment, Sampler: wgpu.SamplerBindingLayout{Type: wgpu.SamplerBindingTypeFiltering}},
		},
	})
	if err != nil {
		postUb.Release()
		postShader.Release()
		releaseUpToEnv()
		return err
	}
	mkBGL := func(label string, extra ...wgpu.BindGroupLayoutEntry) (*wgpu.BindGroupLayout, error) {
		entries := []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageFragment, Texture: wgpu.TextureBindingLayout{SampleType: wgpu.TextureSampleTypeFloat, ViewDimension: wgpu.TextureViewDimension2D}},
			{Binding: 1, Visibility: wgpu.ShaderStageFragment, Sampler: wgpu.SamplerBindingLayout{Type: wgpu.SamplerBindingTypeFiltering}},
		}
		entries = append(entries, extra...)
		return s.device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{Label: label, Entries: entries})
	}
	postBGLBlur, err := mkBGL("post blur BGL",
		wgpu.BindGroupLayoutEntry{Binding: 3, Visibility: wgpu.ShaderStageFragment, Buffer: wgpu.BufferBindingLayout{Type: wgpu.BufferBindingTypeUniform, MinBindingSize: 32}})
	if err != nil {
		postBGLBright.Release()
		postUb.Release()
		postShader.Release()
		releaseUpToEnv()
		return err
	}
	postBGLComposite, err := mkBGL("post composite BGL",
		wgpu.BindGroupLayoutEntry{Binding: 2, Visibility: wgpu.ShaderStageFragment, Texture: wgpu.TextureBindingLayout{SampleType: wgpu.TextureSampleTypeFloat, ViewDimension: wgpu.TextureViewDimension2D}},
		wgpu.BindGroupLayoutEntry{Binding: 3, Visibility: wgpu.ShaderStageFragment, Buffer: wgpu.BufferBindingLayout{Type: wgpu.BufferBindingTypeUniform, MinBindingSize: 32}})
	if err != nil {
		postBGLBlur.Release()
		postBGLBright.Release()
		postUb.Release()
		postShader.Release()
		releaseUpToEnv()
		return err
	}
	mkPlayout := func(bgl *wgpu.BindGroupLayout) (*wgpu.PipelineLayout, error) {
		return s.device.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{
			Label:            "post pipeline layout",
			BindGroupLayouts: []*wgpu.BindGroupLayout{bgl},
		})
	}
	brightPlayout, err := mkPlayout(postBGLBright)
	if err != nil {
		postBGLComposite.Release()
		postBGLBlur.Release()
		postBGLBright.Release()
		postUb.Release()
		postShader.Release()
		releaseUpToEnv()
		return err
	}
	blurPlayout, err := mkPlayout(postBGLBlur)
	if err != nil {
		brightPlayout.Release()
		postBGLComposite.Release()
		postBGLBlur.Release()
		postBGLBright.Release()
		postUb.Release()
		postShader.Release()
		releaseUpToEnv()
		return err
	}
	compositePlayout, err := mkPlayout(postBGLComposite)
	if err != nil {
		blurPlayout.Release()
		brightPlayout.Release()
		postBGLComposite.Release()
		postBGLBlur.Release()
		postBGLBright.Release()
		postUb.Release()
		postShader.Release()
		releaseUpToEnv()
		return err
	}
	brightPl, err := s.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label:       "Bright Pipeline",
		Layout:      brightPlayout,
		Vertex:      wgpu.VertexState{Module: postShader, EntryPoint: "vs_fullscreen"},
		Primitive:   wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		Multisample: wgpu.MultisampleState{Count: 1, Mask: 0xFFFFFFFF},
		Fragment: &wgpu.FragmentState{
			Module:     postShader,
			EntryPoint: "fs_bright",
			Targets: []wgpu.ColorTargetState{
				{Format: wgpu.TextureFormatRGBA16Float, WriteMask: wgpu.ColorWriteMaskAll},
			},
		},
	})
	if err != nil {
		postUb.Release()
		postShader.Release()
		releaseUpToEnv()
		return err
	}
	blurPl, err := s.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label:       "Blur Pipeline",
		Layout:      blurPlayout,
		Vertex:      wgpu.VertexState{Module: postShader, EntryPoint: "vs_fullscreen"},
		Primitive:   wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		Multisample: wgpu.MultisampleState{Count: 1, Mask: 0xFFFFFFFF},
		Fragment: &wgpu.FragmentState{
			Module:     postShader,
			EntryPoint: "fs_blur",
			Targets: []wgpu.ColorTargetState{
				{Format: wgpu.TextureFormatRGBA16Float, WriteMask: wgpu.ColorWriteMaskAll},
			},
		},
	})
	if err != nil {
		brightPl.Release()
		postUb.Release()
		postShader.Release()
		releaseUpToEnv()
		return err
	}
	compositePl, err := s.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label:       "Composite Pipeline",
		Layout:      compositePlayout,
		Vertex:      wgpu.VertexState{Module: postShader, EntryPoint: "vs_fullscreen"},
		Primitive:   wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		Multisample: wgpu.MultisampleState{Count: 1, Mask: 0xFFFFFFFF},
		Fragment: &wgpu.FragmentState{
			Module:     postShader,
			EntryPoint: "fs_composite",
			Targets: []wgpu.ColorTargetState{
				{Format: s.config.Format, WriteMask: wgpu.ColorWriteMaskAll},
			},
		},
	})
	if err != nil {
		blurPl.Release()
		brightPl.Release()
		postUb.Release()
		postShader.Release()
		releaseUpToEnv()
		return err
	}
	postShader.Release()
	brightPlayout.Release()
	blurPlayout.Release()
	compositePlayout.Release()

	s.scene = &sceneState{
		device:                      s.device,
		queue:                       s.queue,
		pipeline:                    pipeline,
		uniformBuffer:               ub,
		bindGroup:                   bg,
		bindGroupShadow:             bgShadow,
		cubeVertexBuf:               cubeBuf,
		cubeVertexCount:             uint32(len(cubeData) / 32),
		meshCache:                   make(map[string]*cachedMesh),
		shadowMap:                   shadowTex,
		shadowView:                  shadowView,
		shadowPipeline:              shadowPl,
		shadowUniform:               shadowUb,
		shadowBindGroup:             sbg,
		shadowSampler:               shadowSampler,
		pointShadowMap:              pointShadowTex,
		pointShadowView:             pointShadowView,
		pointShadowFaces:            pointShadowFaces,
		pointShadowPipeline:         pointShadowPl,
		pointShadowUniform:          pointShadowUb,
		pointShadowBindGroup:        pointShadowBg,
		pointShadowSkinnedPipeline:  pointShadowSkinnedPl,
		pointShadowSkinnedUniform:   pointShadowSkinnedUb,
		pointShadowSkinnedBindGroup: pointShadowSkinnedBg,
		spotShadowMap:               spotShadowTex,
		spotShadowView:              spotShadowView,
		skinnedPipeline:             skinnedPl,
		skinnedUniform:              skinnedUb,
		boneUniform:                 boneUb,
		skinnedBindGroup:            skinnedBg,
		skinnedBindGroupShadow:      skinnedBgShadow,
		meshSkinnedCache:            make(map[string]*cachedMesh),
		shadowSkinnedPipeline:       shadowSkinnedPl,
		shadowSkinnedBindGroup:      shadowSkinnedBg,
		shadowSkinnedUniform:        shadowSkinnedUb,
		envTex:                      envTexR,
		envView:                     envViewR,
		irrTex:                      irrTexR,
		irrView:                     irrViewR,
		envSampler:                  envSmp,
		envBindGroup:                envBg,
		brightPipeline:              brightPl,
		blurPipeline:                blurPl,
		compositePipeline:           compositePl,
		postUniform:                 postUb,
		postBGLBright:               postBGLBright,
		postBGLBlur:                 postBGLBlur,
		postBGLComposite:            postBGLComposite,
	}
	return nil
}

// getOrCreateMeshBuffer возвращает закэшированный vertex buffer для mesh. Создаёт при первом обращении.
func (sc *sceneState) getOrCreateMeshBuffer(resolver *asset.Resolver, meshAssetID string) (*wgpu.Buffer, uint32) {
	if meshAssetID == "" {
		return nil, 0
	}
	if c, ok := sc.meshCache[meshAssetID]; ok && c != nil {
		return c.vertexBuf, c.vertexCount
	}
	positions, normals, uvs, indices := meshFromResolver(resolver, meshAssetID)
	if len(positions) == 0 || len(indices) == 0 {
		return nil, 0
	}
	vertData := buildVertexData(positions, normals, uvs, indices)
	if len(vertData) == 0 {
		return nil, 0
	}
	vb, err := sc.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "mesh cache " + meshAssetID,
		Contents: vertData,
		Usage:    wgpu.BufferUsageVertex,
	})
	if err != nil {
		return nil, 0
	}
	vertexCount := uint32(len(vertData) / 32)
	sc.meshCache[meshAssetID] = &cachedMesh{vertexBuf: vb, vertexCount: vertexCount}
	return vb, vertexCount
}

// getOrCreateSkinnedMeshBuffer возвращает vertex buffer для skinned mesh (stride 64). Создаёт при первом обращении.
func (sc *sceneState) getOrCreateSkinnedMeshBuffer(resolver *asset.Resolver, meshAssetID string) (*wgpu.Buffer, uint32) {
	if meshAssetID == "" || sc.meshSkinnedCache == nil {
		return nil, 0
	}
	if c, ok := sc.meshSkinnedCache[meshAssetID]; ok && c != nil {
		return c.vertexBuf, c.vertexCount
	}
	positions, normals, uvs, joints, weights, indices := meshDataWithSkin(resolver, meshAssetID)
	if len(positions) == 0 || len(indices) == 0 {
		return nil, 0
	}
	vertData := buildVertexDataSkinned(positions, normals, uvs, joints, weights, indices)
	if len(vertData) == 0 {
		return nil, 0
	}
	vb, err := sc.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "skinned mesh " + meshAssetID,
		Contents: vertData,
		Usage:    wgpu.BufferUsageVertex,
	})
	if err != nil {
		return nil, 0
	}
	vertexCount := uint32(len(vertData) / 64)
	sc.meshSkinnedCache[meshAssetID] = &cachedMesh{vertexBuf: vb, vertexCount: vertexCount}
	return vb, vertexCount
}

// pbrSceneData содержит данные для PBR рендера из World.
type pbrSceneData struct {
	viewProj       render.Matrix4
	model          render.Matrix4
	camPos         []float32
	lightDir       []float32
	lightColor     []float32
	lightIntensity float32
	baseColor      []float32
	metallic       float32
	roughness      float32
	ambient        float32

	// Point light (первый в сцене; интенсивность <= 0 — выключен)
	pointLightPos       []float32
	pointLightColor     []float32
	pointLightIntensity float32
	pointLightRange     float32

	// Spotlight (первый в сцене; интенсивность <= 0 — выключен)
	spotLightPos       []float32
	spotLightDir       []float32
	spotLightColor     []float32
	spotLightIntensity float32
	spotLightRange     float32
	spotInnerCos       float32
	spotOuterCos       float32
	spotLightViewProj  render.Matrix4
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

// buildPointFaceViewProj возвращает view-proj для грани cubemap point light.
// Маппинг граней согласован с шейдером (shader.wgsl): 0:+X 1:-X 2:+Y 3:-Y 4:+Z 5:-Z.
func buildPointFaceViewProj(pos emath.Vec3, face int, far float32) render.Matrix4 {
	var dir, up emath.Vec3
	switch face {
	case 0:
		dir, up = emath.Vec3{X: 1, Y: 0, Z: 0}, emath.Vec3{X: 0, Y: -1, Z: 0} // +X
	case 1:
		dir, up = emath.Vec3{X: -1, Y: 0, Z: 0}, emath.Vec3{X: 0, Y: -1, Z: 0} // -X
	case 2:
		dir, up = emath.Vec3{X: 0, Y: 1, Z: 0}, emath.Vec3{X: 0, Y: 0, Z: 1} // +Y
	case 3:
		dir, up = emath.Vec3{X: 0, Y: -1, Z: 0}, emath.Vec3{X: 0, Y: 0, Z: -1} // -Y
	case 4:
		dir, up = emath.Vec3{X: 0, Y: 0, Z: 1}, emath.Vec3{X: 0, Y: -1, Z: 0} // +Z
	case 5:
		dir, up = emath.Vec3{X: 0, Y: 0, Z: -1}, emath.Vec3{X: 0, Y: -1, Z: 0} // -Z
	}
	eye := emath.Vec3{X: pos.X + dir.X, Y: pos.Y + dir.Y, Z: pos.Z + dir.Z}
	view := render.LookAt(pos, eye, up)
	proj := render.Perspective(90, 1, 0.1, far)
	return proj.Multiply(view)
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
		// Конвенция движка (как в orbit_camera.go): forward =
		// (sinY*cosP, -sinP, -cosY*cosP); rotY=0 → взгляд в -Z.
		radX := tr.Rotation.X * math.Pi / 180
		radY := tr.Rotation.Y * math.Pi / 180
		cosP := float32(math.Cos(float64(radX)))
		forward := emath.Vec3{
			X: float32(math.Sin(float64(radY))) * cosP,
			Y: float32(-math.Sin(float64(radX))),
			Z: float32(-math.Cos(float64(radY))) * cosP,
		}
		target := emath.Vec3{
			X: tr.Position.X + forward.X*10,
			Y: tr.Position.Y + forward.Y*10,
			Z: tr.Position.Z + forward.Z*10,
		}
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

	// Первый point light (без теней — cubemap shadows в backlog роадмапа)
	for _, id := range world.Entities() {
		light, hasLight := world.GetLight(id)
		if !hasLight || light.Kind != "point" {
			continue
		}
		tr, hasTr := world.GetTransform(id)
		pos := emath.Vec3{X: 0, Y: 5, Z: 0}
		if hasTr {
			pos = tr.Position
		}
		data.pointLightPos = []float32{pos.X, pos.Y, pos.Z}
		r, g, b := float32(light.ColorR)/255, float32(light.ColorG)/255, float32(light.ColorB)/255
		if r == 0 && g == 0 && b == 0 {
			r, g, b = light.ColorRGB.X, light.ColorRGB.Y, light.ColorRGB.Z
		}
		data.pointLightColor = []float32{r, g, b}
		data.pointLightIntensity = light.Intensity
		data.pointLightRange = light.Range
		if data.pointLightRange <= 0 {
			data.pointLightRange = 10.0
		}
		break
	}

	// Первый spotlight (конус: направление из rotation transform, без теней — в backlog)
	for _, id := range world.Entities() {
		light, hasLight := world.GetLight(id)
		if !hasLight || light.Kind != "spot" {
			continue
		}
		tr, hasTr := world.GetTransform(id)
		pos := emath.Vec3{X: 0, Y: 5, Z: 0}
		dir := emath.Vec3{X: 0, Y: -1, Z: 0}
		if hasTr {
			pos = tr.Position
			radX := tr.Rotation.X * math.Pi / 180
			radY := tr.Rotation.Y * math.Pi / 180
			dir = emath.Vec3{
				X: float32(math.Sin(float64(radY)) * math.Cos(float64(radX))),
				Y: float32(-math.Sin(float64(radX))),
				Z: float32(math.Cos(float64(radY)) * math.Cos(float64(radX))),
			}
		}
		dir = render.Normalize3(dir)
		data.spotLightPos = []float32{pos.X, pos.Y, pos.Z}
		data.spotLightDir = []float32{dir.X, dir.Y, dir.Z}
		r, g, b := float32(light.ColorR)/255, float32(light.ColorG)/255, float32(light.ColorB)/255
		if r == 0 && g == 0 && b == 0 {
			r, g, b = light.ColorRGB.X, light.ColorRGB.Y, light.ColorRGB.Z
		}
		data.spotLightColor = []float32{r, g, b}
		data.spotLightIntensity = light.Intensity
		data.spotLightRange = light.Range
		if data.spotLightRange <= 0 {
			data.spotLightRange = 15.0
		}
		inner := light.SpotInnerAngle
		if inner <= 0 {
			inner = 20
		}
		outer := light.SpotOuterAngle
		if outer <= 0 {
			outer = 30
		}
		if outer < inner {
			outer = inner + 1
		}
		data.spotInnerCos = float32(math.Cos(float64(inner) * math.Pi / 180))
		data.spotOuterCos = float32(math.Cos(float64(outer) * math.Pi / 180))

		// View-proj прожектора для spot-теней (перспектива по внешнему углу конуса).
		up := emath.Vec3{X: 0, Y: 1, Z: 0}
		if abs32(dir.Y) > 0.99 {
			up = emath.Vec3{X: 1, Y: 0, Z: 0}
		}
		view := render.LookAt(pos, emath.Vec3{X: pos.X + dir.X, Y: pos.Y + dir.Y, Z: pos.Z + dir.Z}, up)
		proj := render.Perspective(outer*2, 1, 0.1, data.spotLightRange)
		data.spotLightViewProj = proj.Multiply(view)
		break
	}

	data.ambient = 0.15
	data.metallic = 0.0
	data.roughness = 0.5
	data.baseColor = []float32{0.8, 0.8, 0.8}
	return data, true
}

// materialFlags — биты в materialInfo.flags (uniform flags).
const (
	materialFlagsHasMR = 1 // metallicRoughness текстура задана
)

// materialInfo — параметры материала для uniform-буфера и текстуры.
type materialInfo struct {
	baseColor        []float32
	metallic         float32
	roughness        float32
	emissiveColor    []float32
	emissiveStrength float32
	normalScale      float32
	alphaCutoff      float32
	flags            uint32

	// Пути к .texture.json ("" — текстуры нет, шейдер получит fallback)
	baseColorTex string
	normalTex    string
	mrTex        string
	emissiveTex  string
}

// instanceBatch — группа entities с одинаковым mesh и material для instancing.
type instanceBatch struct {
	meshAssetID string
	materialKey string // MaterialAssetID или "" для default
	transforms  []ecs.Transform
	material    *materialInfo
}

// skinnedDraw — один skinned entity (отдельный draw, свои bone matrices).
type skinnedDraw struct {
	entityID    ecs.EntityID
	meshAssetID string
	transform   ecs.Transform
	material    *materialInfo
}

// isMeshSkinned возвращает true, если mesh имеет skeletal skinning.
func isMeshSkinned(resolver *asset.Resolver, meshAssetID string) bool {
	if resolver == nil || meshAssetID == "" {
		return false
	}
	mesh, err := resolver.ResolveMeshByAssetID(meshAssetID)
	if err != nil || mesh == nil {
		return false
	}
	return mesh.SkinID != "" && len(mesh.Joints) > 0 && len(mesh.Weights) > 0
}

// buildInstanceBatches группирует entities по (mesh, material) для instancing. Skinned meshes исключаются.
func buildInstanceBatches(world *ecs.World, frustum *render.Frustum, camPos emath.Vec3, resolver *asset.Resolver) []instanceBatch {
	group := make(map[string]*instanceBatch)

	for _, id := range world.Entities() {
		mr, hasMR := world.GetMeshRenderer(id)
		if !hasMR {
			continue
		}
		if isMeshSkinned(resolver, mr.MeshAssetID) {
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
		// KENG_NO_CULL=1 — отключить frustum culling (диагностика видимости).
		if os.Getenv("KENG_NO_CULL") == "" && frustum != nil && !frustum.SphereInFrustum(tr.Position, radius) {
			continue
		}
		// Дистанционный culling + LOD по расстоянию до камеры.
		dist := float32(math.Sqrt(float64((tr.Position.X-camPos.X)*(tr.Position.X-camPos.X) +
			(tr.Position.Y-camPos.Y)*(tr.Position.Y-camPos.Y) +
			(tr.Position.Z-camPos.Z)*(tr.Position.Z-camPos.Z))))
		if dist > maxDrawDistance {
			continue
		}
		meshID := selectLODMesh(resolver, mr.MeshAssetID, dist)
		key := meshID + "|" + mr.MaterialAssetID
		if _, ok := group[key]; !ok {
			mat := getMeshMaterial(&mr, resolver)
			group[key] = &instanceBatch{
				meshAssetID: meshID,
				materialKey: mr.MaterialAssetID,
				transforms:  nil,
				material:    mat,
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

// buildSkinnedDraws возвращает список skinned entities для отрисовки.
func buildSkinnedDraws(world *ecs.World, frustum *render.Frustum, camPos emath.Vec3, resolver *asset.Resolver) []skinnedDraw {
	var out []skinnedDraw
	for _, id := range world.Entities() {
		mr, hasMR := world.GetMeshRenderer(id)
		if !hasMR || !isMeshSkinned(resolver, mr.MeshAssetID) {
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
		dx, dy, dz := tr.Position.X-camPos.X, tr.Position.Y-camPos.Y, tr.Position.Z-camPos.Z
		if dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz))); dist > maxDrawDistance {
			continue
		}
		mat := getMeshMaterial(&mr, resolver)
		out = append(out, skinnedDraw{
			entityID:    id,
			meshAssetID: mr.MeshAssetID,
			transform:   tr,
			material:    mat,
		})
	}
	return out
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

// getMeshMaterial собирает параметры материала из MeshRenderer и resolver
// (цвет тинта entity приоритетнее материала; текстуры — только из material).
func getMeshMaterial(mr *ecs.MeshRenderer, resolver *asset.Resolver) *materialInfo {
	mat := &materialInfo{
		baseColor:        []float32{0.8, 0.8, 0.8},
		metallic:         0.0,
		roughness:        0.5,
		emissiveColor:    []float32{0, 0, 0},
		emissiveStrength: 1.0,
		normalScale:      1.0,
		alphaCutoff:      0.5,
	}
	if mr == nil {
		return mat
	}

	if mr.ColorA > 0 {
		mat.baseColor = []float32{
			float32(mr.ColorR) / 255,
			float32(mr.ColorG) / 255,
			float32(mr.ColorB) / 255,
		}
	} else if resolver != nil && mr.MaterialAssetID != "" {
		if m, err := resolver.ResolveMaterialByAssetID(mr.MaterialAssetID); err == nil {
			mat.baseColor = []float32{m.BaseColor.X, m.BaseColor.Y, m.BaseColor.Z}
			mat.metallic = m.Metallic
			mat.roughness = m.Roughness
			mat.emissiveColor = []float32{m.EmissiveColor.X, m.EmissiveColor.Y, m.EmissiveColor.Z}
			mat.emissiveStrength = m.EmissiveStrength
			mat.normalScale = m.NormalScale
			mat.alphaCutoff = m.AlphaCutoff
			mat.baseColorTex = m.BaseColorTex
			mat.normalTex = m.NormalTex
			mat.mrTex = m.MetallicRoughnessTex
			mat.emissiveTex = m.EmissiveTex
			if m.MetallicRoughnessTex != "" {
				mat.flags |= materialFlagsHasMR
			}
		}
	}

	return mat
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
