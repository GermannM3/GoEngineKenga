package asset

// Mesh — промежуточный/derived формат меша для движка (v0: минимум).
type Mesh struct {
	Name string `json:"name"`

	Positions []float32 `json:"positions"` // xyz xyz ...
	Normals   []float32 `json:"normals,omitempty"`
	UV0       []float32 `json:"uv0,omitempty"` // uv uv ...
	Indices   []uint32  `json:"indices"`       // triangle list
}

