package asset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"goenginekenga/engine/asset/gltf"
	"goenginekenga/engine/project"
)

type Record struct {
	ID         string    `json:"id"`
	Type       Type      `json:"type"`
	SourcePath string    `json:"sourcePath"`
	Derived    []string  `json:"derived"`
	ImportedAt time.Time `json:"importedAt"`
}

type Index struct {
	Assets []Record `json:"assets"`
}

type Database struct {
	ProjectDir string
	Project    *project.Project
}

func Open(projectDir string) (*Database, error) {
	p, err := project.Load(projectDir)
	if err != nil {
		return nil, err
	}
	return &Database{ProjectDir: projectDir, Project: p}, nil
}

func (db *Database) assetsDirAbs() string {
	return filepath.Join(db.ProjectDir, filepath.FromSlash(db.Project.AssetsDir))
}

func (db *Database) derivedDirAbs() string {
	return filepath.Join(db.ProjectDir, filepath.FromSlash(db.Project.DerivedDir))
}

func (db *Database) indexPathAbs() string {
	return filepath.Join(db.ProjectDir, ".kenga", "assets", "index.json")
}

func (db *Database) EnsureDirs() error {
	if err := os.MkdirAll(db.derivedDirAbs(), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(db.indexPathAbs()), 0o755); err != nil {
		return err
	}
	return nil
}

func (db *Database) ImportAll() (*Index, error) {
	if err := db.EnsureDirs(); err != nil {
		return nil, err
	}

	root := db.assetsDirAbs()
	var records []Record

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".meta") {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".gltf", ".glb":
			rec, err := db.importGLTF(path)
			if err != nil {
				return err
			}
			records = append(records, *rec)
		default:
			// v0: игнорируем остальные типы
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	idx := &Index{Assets: records}
	if err := db.saveIndex(idx); err != nil {
		return nil, err
	}
	return idx, nil
}

func (db *Database) importGLTF(sourceAbs string) (*Record, error) {
	meta, err := LoadOrCreateMeta(sourceAbs, TypeGLTF)
	if err != nil {
		return nil, err
	}

	res, err := gltf.ImportFile(sourceAbs)
	if err != nil {
		return nil, err
	}
	if len(res.Meshes) == 0 {
		return nil, fmt.Errorf("no meshes imported from %s", sourceAbs)
	}

	derived := make([]string, 0, len(res.Meshes)+len(res.Materials)+len(res.Textures))

	// Сохраняем меши
	for i, m := range res.Meshes {
		name := fmt.Sprintf("%s_%d.mesh.json", meta.ID, i)
		outAbs := filepath.Join(db.derivedDirAbs(), name)
		matIdx := m.MaterialIndex
		if matIdx < 0 || matIdx >= len(res.Materials) {
			matIdx = 0
		}
		matRelPath := filepath.ToSlash(filepath.Join(db.Project.DerivedDir, fmt.Sprintf("%s_%d.material.json", meta.ID, matIdx)))
		mesh := Mesh{
			Name:       m.Name,
			Positions:  m.Positions,
			Normals:    m.Normals,
			UV0:        m.UV0,
			Indices:    m.Indices,
			MaterialID: matRelPath,
		}
		if err := writeJSONFile(outAbs, mesh); err != nil {
			return nil, err
		}
		derived = append(derived, filepath.ToSlash(filepath.Join(db.Project.DerivedDir, name)))
	}

		// Сохраняем материалы
	for i, mat := range res.Materials {
		name := fmt.Sprintf("%s_%d.material.json", meta.ID, i)
		outAbs := filepath.Join(db.derivedDirAbs(), name)
		matCopy := mat
		if i < len(res.BaseColorTexIndex) && res.BaseColorTexIndex[i] >= 0 && res.BaseColorTexIndex[i] < len(res.Textures) {
			texRelPath := filepath.ToSlash(filepath.Join(db.Project.DerivedDir, fmt.Sprintf("%s_%d.texture.json", meta.ID, res.BaseColorTexIndex[i])))
			matCopy.BaseColorTex = texRelPath
		}
		if i < len(res.NormalTexIndex) && res.NormalTexIndex[i] >= 0 && res.NormalTexIndex[i] < len(res.Textures) {
			normRelPath := filepath.ToSlash(filepath.Join(db.Project.DerivedDir, fmt.Sprintf("%s_%d.texture.json", meta.ID, res.NormalTexIndex[i])))
			matCopy.NormalTex = normRelPath
		}
		if err := writeJSONFile(outAbs, matCopy); err != nil {
			return nil, err
		}
		derived = append(derived, filepath.ToSlash(filepath.Join(db.Project.DerivedDir, name)))
	}

	// Сохраняем текстуры
	for i, tex := range res.Textures {
		name := fmt.Sprintf("%s_%d.texture.json", meta.ID, i)
		outAbs := filepath.Join(db.derivedDirAbs(), name)
		// Конвертируем gltf.Texture в asset.Texture
		assetTex := Texture{
			Name:   tex.Name,
			Width:  tex.Width,
			Height: tex.Height,
			Data:   tex.Data,
			Format: tex.Format,
		}
		if err := writeJSONFile(outAbs, assetTex); err != nil {
			return nil, err
		}
		derived = append(derived, filepath.ToSlash(filepath.Join(db.Project.DerivedDir, name)))
	}

	meta.ImportedAt = time.Now()
	if err := SaveMeta(sourceAbs, meta); err != nil {
		return nil, err
	}

	rel, _ := filepath.Rel(db.ProjectDir, sourceAbs)
	return &Record{
		ID:         meta.ID,
		Type:       meta.Type,
		SourcePath: filepath.ToSlash(rel),
		Derived:    derived,
		ImportedAt: meta.ImportedAt,
	}, nil
}

func (db *Database) saveIndex(idx *Index) error {
	return writeJSONFile(db.indexPathAbs(), idx)
}

func writeJSONFile(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
