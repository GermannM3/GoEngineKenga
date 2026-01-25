package asset

import (
	"encoding/json"
	"os"
	"path/filepath"

	"goenginekenga/engine/render"
)

func LoadIndex(projectDir string) (*Index, error) {
	p := filepath.Join(projectDir, ".kenga", "assets", "index.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var idx Index
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

func LoadMesh(path string) (*Mesh, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Mesh
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func LoadMaterial(path string) (*render.Material, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m render.Material
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

