package gltf

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/qmuntal/gltf"
	"github.com/qmuntal/gltf/modeler"

	emath "goenginekenga/engine/math"
	"goenginekenga/engine/render"
)

type ImportResult struct {
	Meshes    []Mesh
	Materials []render.Material
	Textures  []Texture
	// BaseColorTexIndex[i] — индекс текстуры для материала i (-1 если нет)
	BaseColorTexIndex []int
	// NormalTexIndex[i] — индекс normal map для материала i (-1 если нет)
	NormalTexIndex []int
	// Skins — скины для skeletal animation (если есть)
	Skins []SkinData
	// Animations — клипы анимации (узлы и/или скелет)
	Animations []AnimationData
	// NodeNames — имена узлов по индексу (для связи Track.Name с костями)
	NodeNames []string
}

// SkinData — данные скина из glTF (joints, inverse bind matrices)
type SkinData struct {
	Name                string
	Joints              []int           // индексы узлов в doc.Nodes
	JointNames         []string        // имена костей для animation.Track
	ParentIndices      []int           // индекс родительской кости (-1 для root)
	InverseBindMatrices [][16]float32
}

// AnimationData — данные анимации из glTF
type AnimationData struct {
	Name     string
	Duration float32
	Channels []AnimationChannelData
}

// AnimationChannelData — канал анимации (один sampler → один node property)
type AnimationChannelData struct {
	NodeIndex int     // индекс узла в doc.Nodes
	Path      string  // "translation" | "rotation" | "scale"
	Times     []float32
	Values    []float32 // VEC3 или VEC4 (quat для rotation)
}

// Mesh — результат импорта glTF (внутренний для импортера), чтобы избежать циклов импортов.
// Конвертация в `engine/asset.Mesh` делается в AssetDatabase.
type Mesh struct {
	Name          string
	Positions     []float32 // xyz xyz ...
	Normals       []float32 // для освещения
	UV0           []float32 // uv uv ... для текстур
	Indices       []uint32  // triangle list
	MaterialIndex int       // индекс материала в glTF (для связи mesh->material)
	// SkinIndex — индекс в Skins (-1 если меш без скина)
	SkinIndex int
	// Joints — индексы костей на вершину (4 на вершину, [j0,j1,j2,j3] для skinning)
	Joints []uint16
	// Weights — веса костей на вершину (4 на вершину, нормализованы)
	Weights []float32
}

// Texture — результат импорта текстуры из glTF.
type Texture struct {
	Name   string
	Width  int
	Height int
	Data   []byte // RGBA bytes
	Format string // RGBA8, etc.
}

// ImportFile импортирует glTF 2.0 и извлекает минимум (позиции + индексы).
// v0: поддерживает один/несколько мешей; материалы/текстуры будут добавлены позже.
func ImportFile(path string) (*ImportResult, error) {
	doc, err := gltf.Open(path)
	if err != nil {
		return nil, err
	}

	// Подготавливаем буферы
	buffers, err := loadBuffers(doc, filepath.Dir(path))
	if err != nil {
		return nil, err
	}

	// Загружаем текстуры
	var textures []Texture
	for i := range doc.Textures {
		tex, err := loadTextureFromDoc(doc, i, filepath.Dir(path))
		if err != nil {
			// Продолжаем с другими текстурами
			continue
		}
		if tex != nil {
			textures = append(textures, *tex)
		}
	}

	// Загружаем материалы
	var materials []render.Material // nolint
	var baseColorTexIndex []int
	var normalTexIndex []int
	for i, mat := range doc.Materials {
		texIdx := -1
		normIdx := -1
		if mat == nil {
			continue
		}

		material := render.Material{
			Name:             mat.Name,
			Metallic:         0.0,
			Roughness:        0.5,
			AlphaMode:        "OPAQUE",
			AlphaCutoff:      0.5,
			NormalScale:      1.0,
			EmissiveStrength: 1.0,
		}

		if material.Name == "" {
			material.Name = fmt.Sprintf("Material_%d", i)
		}

		// PBR Metallic Roughness
		if mat.PBRMetallicRoughness != nil {
			pbr := mat.PBRMetallicRoughness

			// Base color
			if pbr.BaseColorFactor != nil {
				material.BaseColor = emath.Vec3{
					X: float32(pbr.BaseColorFactor[0]),
					Y: float32(pbr.BaseColorFactor[1]),
					Z: float32(pbr.BaseColorFactor[2]),
				}
			} else {
				material.BaseColor = emath.Vec3{X: 1.0, Y: 1.0, Z: 1.0}
			}

			if pbr.MetallicFactor != nil {
				material.Metallic = float32(*pbr.MetallicFactor)
			}
			if pbr.RoughnessFactor != nil {
				material.Roughness = float32(*pbr.RoughnessFactor)
			}
			if pbr.BaseColorTexture != nil && pbr.BaseColorTexture.Index >= 0 {
				texIdx = pbr.BaseColorTexture.Index
			}
		}

		// Normal texture
		if mat.NormalTexture != nil && mat.NormalTexture.Index != nil && *mat.NormalTexture.Index >= 0 {
			normIdx = *mat.NormalTexture.Index
			if mat.NormalTexture.Scale != nil {
				material.NormalScale = float32(*mat.NormalTexture.Scale)
			}
		}

		baseColorTexIndex = append(baseColorTexIndex, texIdx)
		normalTexIndex = append(normalTexIndex, normIdx)

		// Emissive
		material.EmissiveColor = emath.Vec3{
			X: float32(mat.EmissiveFactor[0]),
			Y: float32(mat.EmissiveFactor[1]),
			Z: float32(mat.EmissiveFactor[2]),
		}

		// Alpha mode
		if mat.AlphaMode != gltf.AlphaOpaque {
			material.AlphaMode = string(mat.AlphaMode)
			if mat.AlphaCutoff != nil {
				material.AlphaCutoff = float32(*mat.AlphaCutoff)
			}
		}

		material.DoubleSided = mat.DoubleSided

		materials = append(materials, material)
	}

	// Если материалов нет, создаем дефолтный
	if len(materials) == 0 {
		materials = append(materials, *render.DefaultMaterial())
	}

	// Имена узлов для анимации
	nodeNames := make([]string, len(doc.Nodes))
	for i, n := range doc.Nodes {
		if n != nil && n.Name != "" {
			nodeNames[i] = n.Name
		} else {
			nodeNames[i] = fmt.Sprintf("Node_%d", i)
		}
	}

	// Карта родительских узлов: nodeIdx -> parentNodeIdx
	parentMap := buildNodeParentMap(doc)

	// Скины
	var skins []SkinData
	for si, skin := range doc.Skins {
		if skin == nil || len(skin.Joints) == 0 {
			continue
		}
		nodeToJoint := make(map[int]int)
		for ji, jIdx := range skin.Joints {
			nodeToJoint[jIdx] = ji
		}
		parentIndices := make([]int, len(skin.Joints))
		for ji, jIdx := range skin.Joints {
			parentNodeIdx := parentMap[jIdx]
			parentIndices[ji] = -1
			if parentNodeIdx >= 0 {
				if pji, ok := nodeToJoint[parentNodeIdx]; ok {
					parentIndices[ji] = pji
				}
			}
		}
		sd := SkinData{
			Name:           skin.Name,
			Joints:         skin.Joints,
			JointNames:     make([]string, len(skin.Joints)),
			ParentIndices:  parentIndices,
		}
		if sd.Name == "" {
			sd.Name = fmt.Sprintf("Skin_%d", si)
		}
		for ji, jIdx := range skin.Joints {
			if jIdx >= 0 && jIdx < len(nodeNames) {
				sd.JointNames[ji] = nodeNames[jIdx]
			} else {
				sd.JointNames[ji] = fmt.Sprintf("Joint_%d", ji)
			}
		}
		if skin.InverseBindMatrices != nil {
			acc := doc.Accessors[*skin.InverseBindMatrices]
			if acc != nil && acc.BufferView != nil {
				matrices, err := readFloat32Mat4(doc, buffers, *skin.InverseBindMatrices)
				if err == nil {
					sd.InverseBindMatrices = matrices
				}
			}
		}
		if len(sd.InverseBindMatrices) == 0 {
			sd.InverseBindMatrices = make([][16]float32, len(skin.Joints))
			for i := range sd.InverseBindMatrices {
				sd.InverseBindMatrices[i] = [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
			}
		}
		skins = append(skins, sd)
	}

	// Анимации
	var animations []AnimationData
	for ai, anim := range doc.Animations {
		if anim == nil || len(anim.Channels) == 0 {
			continue
		}
		ad := AnimationData{
			Name:     anim.Name,
			Channels: make([]AnimationChannelData, 0, len(anim.Channels)),
		}
		if ad.Name == "" {
			ad.Name = fmt.Sprintf("Animation_%d", ai)
		}
		var maxTime float32
		for _, ch := range anim.Channels {
			if ch.Target.Node == nil {
				continue
			}
			nodeIdx := *ch.Target.Node
			samplerIdx := ch.Sampler
			if samplerIdx < 0 || samplerIdx >= len(anim.Samplers) {
				continue
			}
			sampler := anim.Samplers[samplerIdx]
			times, err := readFloat32Scalar(doc, buffers, sampler.Input)
			if err != nil {
				continue
			}
			path := ch.Target.Path.String()
			var values []float32
			switch path {
			case "translation", "scale":
				values, err = readFloat32Vec3(doc, buffers, sampler.Output)
			case "rotation":
				values, err = readFloat32Vec4(doc, buffers, sampler.Output)
			default:
				continue
			}
			if err != nil || len(times) == 0 || len(values) == 0 {
				continue
			}
			if n := len(times); n > 0 && float32(times[n-1]) > maxTime {
				maxTime = float32(times[n-1])
			}
			ad.Channels = append(ad.Channels, AnimationChannelData{
				NodeIndex: nodeIdx,
				Path:      path,
				Times:     times,
				Values:    values,
			})
		}
		ad.Duration = maxTime
		if len(ad.Channels) > 0 {
			animations = append(animations, ad)
		}
	}

	// Построим mesh->node и mesh->skin mapping
	meshToNode := findMeshToNodeMap(doc)
	skinDocToOur := make(map[int]int)
	for si, skin := range doc.Skins {
		if skin == nil || len(skin.Joints) == 0 {
			continue
		}
		skinDocToOur[si] = len(skinDocToOur)
	}
	meshToSkin := make(map[int]int)
	for nodeIdx, meshIdx := range meshToNode {
		if meshIdx < 0 {
			continue
		}
		if n := doc.Nodes[nodeIdx]; n != nil && n.Skin != nil {
			docSkinIdx := *n.Skin
			if ourIdx, ok := skinDocToOur[docSkinIdx]; ok {
				if _, exists := meshToSkin[meshIdx]; !exists {
					meshToSkin[meshIdx] = ourIdx
				}
			}
		}
	}

	var meshes []Mesh
	for mi, m := range doc.Meshes {
		if m == nil {
			continue
		}
		name := m.Name
		if name == "" {
			name = fmt.Sprintf("Mesh_%d", mi)
		}

		// v0: берём только первый primitive каждого mesh, если их несколько — добавим как отдельные меши
		for pi, prim := range m.Primitives {
			if prim == nil {
				continue
			}
			meshName := name
			if pi > 0 {
				meshName = fmt.Sprintf("%s_%d", name, pi)
			}

			posAccIndex, ok := prim.Attributes["POSITION"]
			if !ok {
				continue
			}

			pos, err := readFloat32Vec3(doc, buffers, posAccIndex)
			if err != nil {
				return nil, fmt.Errorf("read POSITION: %w", err)
			}

			var normals, uvs []float32
			if normAcc, ok := prim.Attributes["NORMAL"]; ok {
				normals, _ = readFloat32Vec3(doc, buffers, normAcc)
			}
			if uvAcc, ok := prim.Attributes["TEXCOORD_0"]; ok {
				uvs, _ = readFloat32Vec2(doc, buffers, uvAcc)
			}

			var idx []uint32
			if prim.Indices != nil {
				idx, err = readIndices(doc, buffers, *prim.Indices)
				if err != nil {
					return nil, fmt.Errorf("read Indices: %w", err)
				}
			} else {
				// Если индексов нет — считаем треугольниками подряд.
				idx = make([]uint32, len(pos)/3)
				for i := range idx {
					idx[i] = uint32(i)
				}
			}

			matIdx := 0
			if prim.Material != nil && *prim.Material >= 0 {
				matIdx = *prim.Material
			}

			skinIdx := -1
			var jData []uint16
			var wData []float32
			if sIdx, ok := meshToSkin[mi]; ok {
				skinIdx = sIdx
				if jointsAcc, ok := prim.Attributes[gltf.JOINTS_0]; ok {
					jData, _ = readJoints(doc, buffers, jointsAcc)
				}
				if weightsAcc, ok := prim.Attributes[gltf.WEIGHTS_0]; ok {
					wData, _ = readWeights(doc, buffers, weightsAcc)
				}
			}

			meshes = append(meshes, Mesh{
				Name:          meshName,
				Positions:     pos,
				Normals:       normals,
				UV0:           uvs,
				Indices:       idx,
				MaterialIndex: matIdx,
				SkinIndex:     skinIdx,
				Joints:        jData,
				Weights:       wData,
			})
		}
	}

	return &ImportResult{
		Meshes:            meshes,
		Materials:         materials,
		Textures:          textures,
		BaseColorTexIndex: baseColorTexIndex,
		NormalTexIndex:    normalTexIndex,
		Skins:             skins,
		Animations:        animations,
		NodeNames:         nodeNames,
	}, nil
}

// buildNodeParentMap возвращает map: nodeIdx -> parentNodeIdx (-1 если root)
func buildNodeParentMap(doc *gltf.Document) map[int]int {
	out := make(map[int]int)
	var visit func(nodeIdx int, parent int)
	visit = func(nodeIdx int, parent int) {
		if nodeIdx < 0 || nodeIdx >= len(doc.Nodes) {
			return
		}
		n := doc.Nodes[nodeIdx]
		if n == nil {
			return
		}
		out[nodeIdx] = parent
		for _, c := range n.Children {
			visit(c, nodeIdx)
		}
	}
	sceneIdx := 0
	if doc.Scene != nil {
		sceneIdx = *doc.Scene
	}
	if sceneIdx >= 0 && sceneIdx < len(doc.Scenes) {
		for _, nodeIdx := range doc.Scenes[sceneIdx].Nodes {
			visit(nodeIdx, -1)
		}
	}
	return out
}

// findMeshToNodeMap возвращает map: nodeIndex -> meshIndex (первый узел со ссылкой на меш)
func findMeshToNodeMap(doc *gltf.Document) map[int]int {
	out := make(map[int]int)
	var visit func(nodeIdx int)
	visit = func(nodeIdx int) {
		if nodeIdx < 0 || nodeIdx >= len(doc.Nodes) {
			return
		}
		n := doc.Nodes[nodeIdx]
		if n == nil {
			return
		}
		if n.Mesh != nil {
			mi := *n.Mesh
			if mi >= 0 && mi < len(doc.Meshes) {
				out[nodeIdx] = mi
			}
		}
		for _, c := range n.Children {
			visit(c)
		}
	}
	sceneIdx := 0
	if doc.Scene != nil {
		sceneIdx = *doc.Scene
	}
	if sceneIdx >= 0 && sceneIdx < len(doc.Scenes) {
		for _, nodeIdx := range doc.Scenes[sceneIdx].Nodes {
			visit(nodeIdx)
		}
	}
	return out
}

func readFloat32Mat4(doc *gltf.Document, buffers [][]byte, accessorIndex int) ([][16]float32, error) {
	if accessorIndex < 0 || accessorIndex >= len(doc.Accessors) {
		return nil, fmt.Errorf("accessor %d out of range", accessorIndex)
	}
	acc := doc.Accessors[accessorIndex]
	if acc == nil || acc.BufferView == nil {
		return nil, fmt.Errorf("accessor %d invalid", accessorIndex)
	}
	if acc.Type != gltf.AccessorMat4 {
		return nil, fmt.Errorf("expected MAT4, got %s", acc.Type)
	}
	bvIndex := int(*acc.BufferView)
	if bvIndex < 0 || bvIndex >= len(doc.BufferViews) {
		return nil, fmt.Errorf("bufferView %d out of range", bvIndex)
	}
	view := doc.BufferViews[bvIndex]
	bufIndex := int(view.Buffer)
	if bufIndex < 0 || bufIndex >= len(buffers) {
		return nil, fmt.Errorf("buffer %d out of range", bufIndex)
	}
	buf := buffers[bufIndex]
	byteOffset := int(view.ByteOffset) + int(acc.ByteOffset)
	byteStride := int(view.ByteStride)
	if byteStride == 0 {
		byteStride = 64
	}
	count := int(acc.Count)
	out := make([][16]float32, count)
	for i := 0; i < count; i++ {
		off := byteOffset + i*byteStride
		if off+64 > len(buf) {
			return nil, fmt.Errorf("buffer OOB reading MAT4")
		}
		for j := 0; j < 16; j++ {
			out[i][j] = mathFloat32LE(buf[off+j*4 : off+j*4+4])
		}
	}
	return out, nil
}

func readFloat32Scalar(doc *gltf.Document, buffers [][]byte, accessorIndex int) ([]float32, error) {
	if accessorIndex < 0 || accessorIndex >= len(doc.Accessors) {
		return nil, fmt.Errorf("accessor %d out of range", accessorIndex)
	}
	acc := doc.Accessors[accessorIndex]
	if acc == nil || acc.BufferView == nil {
		return nil, fmt.Errorf("accessor %d invalid", accessorIndex)
	}
	if acc.Type != gltf.AccessorScalar {
		return nil, fmt.Errorf("expected SCALAR, got %s", acc.Type)
	}
	bvIndex := int(*acc.BufferView)
	if bvIndex < 0 || bvIndex >= len(doc.BufferViews) {
		return nil, fmt.Errorf("bufferView %d out of range", bvIndex)
	}
	view := doc.BufferViews[bvIndex]
	bufIndex := int(view.Buffer)
	if bufIndex < 0 || bufIndex >= len(buffers) {
		return nil, fmt.Errorf("buffer %d out of range", bufIndex)
	}
	buf := buffers[bufIndex]
	byteOffset := int(view.ByteOffset) + int(acc.ByteOffset)
	byteStride := int(view.ByteStride)
	if byteStride == 0 {
		byteStride = 4
	}
	count := int(acc.Count)
	out := make([]float32, count)
	for i := 0; i < count; i++ {
		off := byteOffset + i*byteStride
		if off+4 > len(buf) {
			return nil, fmt.Errorf("buffer OOB reading SCALAR")
		}
		out[i] = mathFloat32LE(buf[off : off+4])
	}
	return out, nil
}

func readFloat32Vec4(doc *gltf.Document, buffers [][]byte, accessorIndex int) ([]float32, error) {
	if accessorIndex < 0 || accessorIndex >= len(doc.Accessors) {
		return nil, fmt.Errorf("accessor %d out of range", accessorIndex)
	}
	acc := doc.Accessors[accessorIndex]
	if acc == nil || acc.BufferView == nil {
		return nil, fmt.Errorf("accessor %d invalid", accessorIndex)
	}
	if acc.ComponentType != gltf.ComponentFloat {
		return nil, fmt.Errorf("expected float component type, got %d", acc.ComponentType)
	}
	if acc.Type != gltf.AccessorVec4 {
		return nil, fmt.Errorf("expected VEC4, got %s", acc.Type)
	}
	bvIndex := int(*acc.BufferView)
	if bvIndex < 0 || bvIndex >= len(doc.BufferViews) {
		return nil, fmt.Errorf("bufferView %d out of range", bvIndex)
	}
	view := doc.BufferViews[bvIndex]
	bufIndex := int(view.Buffer)
	if bufIndex < 0 || bufIndex >= len(buffers) {
		return nil, fmt.Errorf("buffer %d out of range", bufIndex)
	}
	buf := buffers[bufIndex]
	byteOffset := int(view.ByteOffset) + int(acc.ByteOffset)
	byteStride := int(view.ByteStride)
	if byteStride == 0 {
		byteStride = 16
	}
	count := int(acc.Count)
	out := make([]float32, 0, count*4)
	for i := 0; i < count; i++ {
		off := byteOffset + i*byteStride
		if off+16 > len(buf) {
			return nil, fmt.Errorf("buffer OOB reading VEC4")
		}
		out = append(out,
			mathFloat32LE(buf[off:off+4]),
			mathFloat32LE(buf[off+4:off+8]),
			mathFloat32LE(buf[off+8:off+12]),
			mathFloat32LE(buf[off+12:off+16]),
		)
	}
	return out, nil
}

func readJoints(doc *gltf.Document, buffers [][]byte, accessorIndex int) ([]uint16, error) {
	acc := doc.Accessors[accessorIndex]
	if acc == nil || acc.BufferView == nil {
		return nil, fmt.Errorf("accessor invalid")
	}
	var jBuf [][4]uint16
	data, err := modeler.ReadJoints(doc, acc, jBuf)
	if err != nil {
		return nil, err
	}
	flat := make([]uint16, 0, len(data)*4)
	for _, v := range data {
		flat = append(flat, v[0], v[1], v[2], v[3])
	}
	return flat, nil
}

func readWeights(doc *gltf.Document, buffers [][]byte, accessorIndex int) ([]float32, error) {
	acc := doc.Accessors[accessorIndex]
	if acc == nil || acc.BufferView == nil {
		return nil, fmt.Errorf("accessor invalid")
	}
	var wBuf [][4]float32
	data, err := modeler.ReadWeights(doc, acc, wBuf)
	if err != nil {
		return nil, err
	}
	flat := make([]float32, 0, len(data)*4)
	for _, v := range data {
		flat = append(flat, v[0], v[1], v[2], v[3])
	}
	return flat, nil
}

func loadBuffers(doc *gltf.Document, baseDir string) ([][]byte, error) {
	out := make([][]byte, len(doc.Buffers))
	for i, b := range doc.Buffers {
		if b == nil {
			continue
		}
		switch {
		case b.URI == "":
			// GLB: бинарные данные должны быть в doc.Buffers[i].Data
			if len(b.Data) == 0 {
				return nil, fmt.Errorf("buffer %d has no uri and no data", i)
			}
			out[i] = b.Data
		case strings.HasPrefix(b.URI, "data:"):
			data, err := decodeDataURI(b.URI)
			if err != nil {
				return nil, err
			}
			out[i] = data
		default:
			p := filepath.Join(baseDir, filepath.FromSlash(b.URI))
			data, err := os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			out[i] = data
		}
	}
	return out, nil
}

func readFloat32Vec3(doc *gltf.Document, buffers [][]byte, accessorIndex int) ([]float32, error) {
	if accessorIndex < 0 || accessorIndex >= len(doc.Accessors) {
		return nil, fmt.Errorf("accessor %d out of range", accessorIndex)
	}
	acc := doc.Accessors[accessorIndex]
	if acc == nil || acc.BufferView == nil {
		return nil, fmt.Errorf("accessor %d invalid", accessorIndex)
	}
	if acc.ComponentType != gltf.ComponentFloat {
		return nil, fmt.Errorf("expected float component type, got %d", acc.ComponentType)
	}
	if acc.Type != gltf.AccessorVec3 {
		return nil, fmt.Errorf("expected VEC3, got %s", acc.Type)
	}

	bvIndex := int(*acc.BufferView)
	if bvIndex < 0 || bvIndex >= len(doc.BufferViews) {
		return nil, fmt.Errorf("bufferView %d out of range", bvIndex)
	}
	view := doc.BufferViews[bvIndex]
	if view == nil {
		return nil, fmt.Errorf("bufferView %d is nil", *acc.BufferView)
	}
	bufIndex := int(view.Buffer)
	if bufIndex < 0 || bufIndex >= len(buffers) {
		return nil, fmt.Errorf("buffer %d out of range", bufIndex)
	}
	buf := buffers[bufIndex]

	byteOffset := int(view.ByteOffset) + int(acc.ByteOffset)
	byteStride := int(view.ByteStride)
	if byteStride == 0 {
		byteStride = 12 // 3 * float32
	}

	count := int(acc.Count)
	out := make([]float32, 0, count*3)
	for i := 0; i < count; i++ {
		off := byteOffset + i*byteStride
		if off+12 > len(buf) {
			return nil, fmt.Errorf("buffer OOB reading VEC3")
		}
		out = append(out,
			mathFloat32LE(buf[off:off+4]),
			mathFloat32LE(buf[off+4:off+8]),
			mathFloat32LE(buf[off+8:off+12]),
		)
	}
	return out, nil
}

func readFloat32Vec2(doc *gltf.Document, buffers [][]byte, accessorIndex int) ([]float32, error) {
	if accessorIndex < 0 || accessorIndex >= len(doc.Accessors) {
		return nil, fmt.Errorf("accessor %d out of range", accessorIndex)
	}
	acc := doc.Accessors[accessorIndex]
	if acc == nil || acc.BufferView == nil {
		return nil, fmt.Errorf("accessor %d invalid", accessorIndex)
	}
	if acc.ComponentType != gltf.ComponentFloat {
		return nil, fmt.Errorf("expected float component type, got %d", acc.ComponentType)
	}
	if acc.Type != gltf.AccessorVec2 {
		return nil, fmt.Errorf("expected VEC2, got %s", acc.Type)
	}
	bvIndex := int(*acc.BufferView)
	if bvIndex < 0 || bvIndex >= len(doc.BufferViews) {
		return nil, fmt.Errorf("bufferView %d out of range", bvIndex)
	}
	view := doc.BufferViews[bvIndex]
	if view == nil {
		return nil, fmt.Errorf("bufferView %d is nil", *acc.BufferView)
	}
	bufIndex := int(view.Buffer)
	if bufIndex < 0 || bufIndex >= len(buffers) {
		return nil, fmt.Errorf("buffer %d out of range", bufIndex)
	}
	buf := buffers[bufIndex]
	byteOffset := int(view.ByteOffset) + int(acc.ByteOffset)
	byteStride := int(view.ByteStride)
	if byteStride == 0 {
		byteStride = 8
	}
	count := int(acc.Count)
	out := make([]float32, 0, count*2)
	for i := 0; i < count; i++ {
		off := byteOffset + i*byteStride
		if off+8 > len(buf) {
			return nil, fmt.Errorf("buffer OOB reading VEC2")
		}
		out = append(out,
			mathFloat32LE(buf[off:off+4]),
			mathFloat32LE(buf[off+4:off+8]),
		)
	}
	return out, nil
}

func readIndices(doc *gltf.Document, buffers [][]byte, accessorIndex int) ([]uint32, error) {
	if accessorIndex < 0 || accessorIndex >= len(doc.Accessors) {
		return nil, fmt.Errorf("accessor %d out of range", accessorIndex)
	}
	acc := doc.Accessors[accessorIndex]
	if acc == nil || acc.BufferView == nil {
		return nil, fmt.Errorf("accessor %d invalid", accessorIndex)
	}
	if acc.Type != gltf.AccessorScalar {
		return nil, fmt.Errorf("expected SCALAR indices, got %s", acc.Type)
	}
	bvIndex := int(*acc.BufferView)
	if bvIndex < 0 || bvIndex >= len(doc.BufferViews) {
		return nil, fmt.Errorf("bufferView %d out of range", bvIndex)
	}
	view := doc.BufferViews[bvIndex]
	if view == nil {
		return nil, fmt.Errorf("bufferView %d is nil", *acc.BufferView)
	}
	bufIndex := int(view.Buffer)
	if bufIndex < 0 || bufIndex >= len(buffers) {
		return nil, fmt.Errorf("buffer %d out of range", bufIndex)
	}
	buf := buffers[bufIndex]

	byteOffset := int(view.ByteOffset) + int(acc.ByteOffset)
	byteStride := int(view.ByteStride)
	if byteStride == 0 {
		byteStride = componentSize(acc.ComponentType)
	}

	count := int(acc.Count)
	out := make([]uint32, 0, count)
	for i := 0; i < count; i++ {
		off := byteOffset + i*byteStride
		if off+componentSize(acc.ComponentType) > len(buf) {
			return nil, fmt.Errorf("buffer OOB reading indices")
		}
		switch acc.ComponentType {
		case gltf.ComponentUshort:
			out = append(out, uint32(binary.LittleEndian.Uint16(buf[off:off+2])))
		case gltf.ComponentUint:
			out = append(out, binary.LittleEndian.Uint32(buf[off:off+4]))
		case gltf.ComponentUbyte:
			out = append(out, uint32(buf[off]))
		default:
			return nil, fmt.Errorf("unsupported index component type %d", acc.ComponentType)
		}
	}
	return out, nil
}

func componentSize(ct gltf.ComponentType) int {
	switch ct {
	case gltf.ComponentUbyte:
		return 1
	case gltf.ComponentUshort:
		return 2
	case gltf.ComponentUint:
		return 4
	case gltf.ComponentFloat:
		return 4
	default:
		return 4
	}
}

func mathFloat32LE(b []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b))
}

// decodeDataURI декодирует data: URI (поддерживает base64).
// Пример: data:application/octet-stream;base64,AAAA...
func decodeDataURI(uri string) ([]byte, error) {
	if !strings.HasPrefix(uri, "data:") {
		return nil, fmt.Errorf("not a data uri")
	}
	// data:[<mediatype>][;base64],<data>
	comma := strings.IndexByte(uri, ',')
	if comma < 0 {
		return nil, fmt.Errorf("invalid data uri: no comma")
	}
	meta := uri[:comma]
	dataPart := uri[comma+1:]

	if strings.Contains(meta, ";base64") {
		b, err := base64.StdEncoding.DecodeString(dataPart)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
	// v0: без percent-decoding (редко в glTF). Если понадобится — добавим.
	return []byte(dataPart), nil
}

// loadTextureFromDoc загружает текстуру из glTF документа
func loadTextureFromDoc(doc *gltf.Document, textureIndex int, baseDir string) (*Texture, error) {
	if textureIndex < 0 || textureIndex >= len(doc.Textures) {
		return nil, nil
	}

	tex := doc.Textures[textureIndex]
	if tex == nil || tex.Source == nil {
		return nil, nil
	}

	sourceIndex := int(*tex.Source)
	if sourceIndex < 0 || sourceIndex >= len(doc.Images) {
		return nil, nil
	}

	img := doc.Images[sourceIndex]
	if img == nil {
		return nil, nil
	}

	var imagePath string
	if img.URI != "" {
		if strings.HasPrefix(img.URI, "data:") {
			return nil, nil // data URI пока не поддерживаем
		}
		imagePath = filepath.Join(baseDir, filepath.FromSlash(img.URI))
	} else if img.BufferView != nil {
		return nil, nil // GLB buffer пока не поддерживаем
	} else {
		return nil, nil
	}

	if imagePath == "" {
		return nil, nil
	}

	return loadTextureFromFile(imagePath)
}

// loadTextureFromFile загружает текстуру из файла
func loadTextureFromFile(path string) (*Texture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	return &Texture{
		Name:   filepath.Base(path),
		Width:  width,
		Height: height,
		Data:   rgba.Pix,
		Format: "RGBA8",
	}, nil
}
