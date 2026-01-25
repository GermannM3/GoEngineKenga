package asset

import (
	"image"
	_ "image/jpeg" // поддержка JPEG
	_ "image/png"  // поддержка PNG
	"os"
	"path/filepath"
)

// Texture представляет загруженную текстуру
type Texture struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Data   []byte `json:"data"`   // RGBA bytes
	Format string `json:"format"` // RGBA8, etc.
}

// TextureAsset представляет asset текстуры с метаданными
type TextureAsset struct {
	*Texture
	AssetID string `json:"assetId"`
	Path    string `json:"path"`
}

// LoadTextureFromFile загружает текстуру из файла
func LoadTextureFromFile(path string) (*Texture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Конвертируем в RGBA
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	return &Texture{
		Name:   filepath.Base(path),
		Width:  width,
		Height: height,
		Data:   rgba.Pix,
		Format: "RGBA8",
	}, nil
}
