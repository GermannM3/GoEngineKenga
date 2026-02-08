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

// Refresh перезагружает индекс ассетов с диска. Вызывать после re-import.
func (r *Resolver) Refresh() error {
	idx, err := LoadIndex(r.projectDir)
	if err != nil {
		return err
	}
	r.index = idx
	r.byID = make(map[string]Record)
	for _, rec := range idx.Assets {
		r.byID[rec.ID] = rec
	}
	return nil
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

// ResolveMeshByPath загружает меш по относительному пути (например ".kenga/derived/xxx_0.mesh.json").
func (r *Resolver) ResolveMeshByPath(relPath string) (*Mesh, error) {
	if relPath == "" {
		return nil, fmt.Errorf("mesh path is empty")
	}
	abs := filepath.Join(r.projectDir, filepath.FromSlash(relPath))
	return LoadMesh(abs)
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

// ResolveMaterialByPath загружает материал по относительному пути (например "derived/xxx_0.material.json")
func (r *Resolver) ResolveMaterialByPath(relPath string) (*render.Material, error) {
	if relPath == "" {
		return nil, fmt.Errorf("material path is empty")
	}
	abs := filepath.Join(r.projectDir, filepath.FromSlash(relPath))
	return LoadMaterial(abs)
}

// ResolveTextureByPath загружает текстуру по относительному пути (например "derived/xxx_0.texture.json")
func (r *Resolver) ResolveTextureByPath(relPath string) (*Texture, error) {
	if relPath == "" {
		return nil, fmt.Errorf("texture path is empty")
	}
	abs := filepath.Join(r.projectDir, filepath.FromSlash(relPath))
	return LoadTexture(abs)
}
