package asset

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeGLTF     Type = "gltf"
	TypeMesh     Type = "mesh"
	TypeMaterial Type = "material"
	TypeTexture  Type = "texture"
	TypeWASM     Type = "wasm"
	TypeAudio    Type = "audio"
)

type Meta struct {
	ID         string    `json:"id"`
	Type       Type      `json:"type"`
	SourcePath string    `json:"sourcePath"`
	ImportedAt time.Time `json:"importedAt"`
}

func metaPathForSource(sourcePath string) string {
	return sourcePath + ".meta"
}

func LoadOrCreateMeta(sourcePath string, typ Type) (*Meta, error) {
	mp := metaPathForSource(sourcePath)
	if b, err := os.ReadFile(mp); err == nil {
		var m Meta
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, err
		}
		// Если тип изменился — обновим.
		if m.Type == "" {
			m.Type = typ
		}
		if m.SourcePath == "" {
			m.SourcePath = filepath.ToSlash(sourcePath)
		}
		if m.ID == "" {
			m.ID = uuid.NewString()
		}
		return &m, nil
	}

	m := &Meta{
		ID:         uuid.NewString(),
		Type:       typ,
		SourcePath: filepath.ToSlash(sourcePath),
		ImportedAt: time.Time{},
	}
	if err := SaveMeta(sourcePath, m); err != nil {
		return nil, err
	}
	return m, nil
}

func SaveMeta(sourcePath string, m *Meta) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPathForSource(sourcePath), b, 0o644)
}
