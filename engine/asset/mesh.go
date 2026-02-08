package asset

// Mesh — промежуточный/derived формат меша для движка (v0: минимум).
type Mesh struct {
	Name string `json:"name"`

	Positions []float32 `json:"positions"` // xyz xyz ...
	Normals   []float32 `json:"normals,omitempty"`
	UV0       []float32 `json:"uv0,omitempty"` // uv uv ...
	Indices   []uint32  `json:"indices"`       // triangle list

	// MaterialID — путь к .material.json (относительно проекта), для текстур и PBR
	MaterialID string `json:"materialId,omitempty"`

	// LODRefs — пути к упрощённым мешам (LOD1, LOD2, ...), относительно проекта.
	// LOD0 = этот меш. При расстоянии > LODThreshold рендерер переключается на LODRefs[0] и т.д.
	LODRefs []string `json:"lodRefs,omitempty"`
}
