package gltf

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/qmuntal/gltf"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/render"
)

type ImportResult struct {
	Meshes    []Mesh
	Materials []Material
	Textures  []Texture
}

// Mesh — результат импорта glTF (внутренний для импортера), чтобы избежать циклов импортов.
// Конвертация в `engine/asset.Mesh` делается в AssetDatabase.
type Mesh struct {
	Name      string
	Positions []float32 // xyz xyz ...
	Indices   []uint32  // triangle list
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
	var textures []asset.Texture
	for i, img := range doc.Images {
		if img == nil {
			continue
		}

		tex, err := asset.LoadTextureFromGLTF(doc, i, filepath.Dir(path))
		if err != nil {
			// Продолжаем с другими текстурами
			continue
		}
		if tex != nil {
			textures = append(textures, *tex)
		}
	}

	// Загружаем материалы
	var materials []render.Material
	for i, mat := range doc.Materials {
		if mat == nil {
			continue
		}

		material := render.Material{
			Name:        mat.Name,
			Metallic:    0.0,
			Roughness:   0.5,
			AlphaMode:   "OPAQUE",
			AlphaCutoff: 0.5,
			NormalScale: 1.0,
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
				material.BaseColor = [3]float32{pbr.BaseColorFactor[0], pbr.BaseColorFactor[1], pbr.BaseColorFactor[2]}
			} else {
				material.BaseColor = [3]float32{1.0, 1.0, 1.0}
			}

			material.Metallic = float32(pbr.MetallicFactor)
			material.Roughness = float32(pbr.RoughnessFactor)

			// Textures (пока пропустим, добавим позже)
		}

		// Emissive
		if mat.EmissiveFactor != nil {
			material.EmissiveColor = [3]float32{mat.EmissiveFactor[0], mat.EmissiveFactor[1], mat.EmissiveFactor[2]}
		}

		// Alpha mode
		if mat.AlphaMode != gltf.AlphaOpaque {
			material.AlphaMode = string(mat.AlphaMode)
			material.AlphaCutoff = float32(mat.AlphaCutoff)
		}

		material.DoubleSided = mat.DoubleSided

		materials = append(materials, material)
	}

	// Если материалов нет, создаем дефолтный
	if len(materials) == 0 {
		materials = append(materials, *render.DefaultMaterial())
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

			meshes = append(meshes, Mesh{
				Name:      meshName,
				Positions: pos,
				Indices:   idx,
			})
		}
	}

	return &ImportResult{
		Meshes:    meshes,
		Materials: materials,
		Textures:  textures,
	}, nil
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

