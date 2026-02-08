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

// LoadTexture загружает текстуру из .texture.json (Data в JSON — base64)
func LoadTexture(path string) (*Texture, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Texture
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// AudioClipAsset represents an audio clip with metadata
type AudioClipAsset struct {
	Name   string `json:"name"`
	Format string `json:"format"` // wav, mp3, ogg
	Data   []byte `json:"data"`
}

// LoadAudioClip loads an audio file from path
func LoadAudioClip(path string) (*AudioClipAsset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Determine format from extension
	ext := filepath.Ext(path)
	format := "wav"
	switch ext {
	case ".mp3":
		format = "mp3"
	case ".ogg":
		format = "ogg"
	case ".wav":
		format = "wav"
	}

	return &AudioClipAsset{
		Name:   filepath.Base(path),
		Format: format,
		Data:   data,
	}, nil
}
