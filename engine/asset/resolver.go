package asset

import (
	"fmt"
	"path/filepath"
	"strings"

	"goenginekenga/engine/render"
)

type Resolver struct {
	projectDir string
	index      *Index
	byID       map[string]Record
}

func NewResolver(projectDir string) (*Resolver, error) {
	idx, err := LoadIndex(projectDir)
	if err != nil {
		return nil, err
	}
	r := &Resolver{
		projectDir: projectDir,
		index:      idx,
		byID:       map[string]Record{},
	}
	for _, rec := range idx.Assets {
		r.byID[rec.ID] = rec
	}
	return r, nil
}

func (r *Resolver) ResolveMeshByAssetID(assetID string) (*Mesh, error) {
	rec, ok := r.byID[assetID]
	if !ok {
		return nil, fmt.Errorf("asset id not found: %s", assetID)
	}
	for _, d := range rec.Derived {
		if strings.HasSuffix(d, ".mesh.json") {
			abs := filepath.Join(r.projectDir, filepath.FromSlash(d))
			return LoadMesh(abs)
		}
	}
	return nil, fmt.Errorf("no derived mesh for asset id: %s", assetID)
}

func (r *Resolver) ResolveMaterialByAssetID(assetID string) (*render.Material, error) {
	rec, ok := r.byID[assetID]
	if !ok {
		return nil, fmt.Errorf("asset id not found: %s", assetID)
	}
	for _, d := range rec.Derived {
		if strings.HasSuffix(d, ".material.json") {
			abs := filepath.Join(r.projectDir, filepath.FromSlash(d))
			return LoadMaterial(abs)
		}
	}
	return nil, fmt.Errorf("no derived material for asset id: %s", assetID)
}

