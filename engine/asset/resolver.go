package asset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goenginekenga/engine/audio"
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

// ResolveSkeletonByPath загружает скелет по относительному пути (например ".kenga/derived/xxx_skin_0.skeleton.json").
func (r *Resolver) ResolveSkeletonByPath(relPath string) (*Skeleton, error) {
	if relPath == "" {
		return nil, fmt.Errorf("skeleton path is empty")
	}
	abs := filepath.Join(r.projectDir, filepath.FromSlash(relPath))
	return LoadSkeleton(abs)
}

// ResolveAudioClipBySource возвращает asset ID клипа по пути исходника
// (например "assets/audio/hit.wav"). "" если не найден.
func (r *Resolver) ResolveAudioClipBySource(relPath string) string {
	return r.AssetIDBySource(relPath)
}

// AssetIDBySource возвращает asset ID любого ассета по относительному пути
// исходника ("assets/meshes/crystal.mesh.json"). "" если не найден.
func (r *Resolver) AssetIDBySource(relPath string) string {
	rel := filepath.ToSlash(relPath)
	for _, rec := range r.index.Assets {
		if strings.HasSuffix(rec.SourcePath, rel) {
			return rec.ID
		}
	}
	return ""
}

// ResolveAudioClip загружает аудиоклип по asset ID (wav/mp3/ogg).
func (r *Resolver) ResolveAudioClip(assetID string) (*audio.AudioClip, error) {
	rec, ok := r.byID[assetID]
	if !ok {
		return nil, fmt.Errorf("asset id not found: %s", assetID)
	}
	if rec.Type != TypeAudio || rec.SourcePath == "" {
		return nil, fmt.Errorf("asset %s is not audio", assetID)
	}
	abs := filepath.Join(r.projectDir, filepath.FromSlash(rec.SourcePath))
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(abs)), ".")
	if format != "wav" && format != "mp3" && format != "ogg" {
		format = "wav"
	}
	return audio.LoadAudioClip(rec.SourcePath, data, format), nil
}
