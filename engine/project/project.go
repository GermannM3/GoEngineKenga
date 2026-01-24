package project

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Project struct {
	Name       string   `json:"name"`
	Scenes     []string `json:"scenes"`
	AssetsDir  string   `json:"assetsDir"`
	DerivedDir string   `json:"derivedDir"`
}

func Load(dir string) (*Project, error) {
	b, err := os.ReadFile(filepath.Join(dir, "project.kenga.json"))
	if err != nil {
		return nil, err
	}
	var p Project
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	if p.AssetsDir == "" {
		p.AssetsDir = "assets"
	}
	if p.DerivedDir == "" {
		p.DerivedDir = ".kenga/derived"
	}
	return &p, nil
}

