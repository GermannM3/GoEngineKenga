package asset

// Skeleton — данные скелета для skeletal animation (из glTF Skin).
type Skeleton struct {
	Name               string     `json:"name"`
	JointNames         []string   `json:"jointNames"`
	ParentIndices      []int      `json:"parentIndices"` // -1 для root
	InverseBindMatrices [][16]float32 `json:"inverseBindMatrices"`
}
