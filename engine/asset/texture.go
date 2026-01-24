package asset

import (
	"image"
	_ "image/jpeg" // поддержка JPEG
	_ "image/png"  // поддержка PNG
	"os"
	"path/filepath"

	"github.com/qmuntal/gltf"
)

// Texture представляет загруженную текстуру
type Texture struct {
	Name   string      `json:"name"`
	Width  int         `json:"width"`
	Height int         `json:"height"`
	Data   []byte      `json:"data"`   // RGBA bytes
	Format string      `json:"format"` // RGBA8, etc.
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

	img, format, err := image.Decode(file)
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

// LoadTextureFromGLTF загружает текстуру из glTF документа
func LoadTextureFromGLTF(doc *gltf.Document, textureIndex int, baseDir string) (*Texture, error) {
	if textureIndex < 0 || textureIndex >= len(doc.Textures) {
		return nil, nil // текстура не указана
	}

	tex := doc.Textures[textureIndex]
	if tex == nil {
		return nil, nil
	}

	// В glTF текстуры ссылаются на источники изображений
	if tex.Source == nil {
		return nil, nil
	}

	sourceIndex := int(*tex.Source)
	if sourceIndex < 0 || sourceIndex >= len(doc.Images) {
		return nil, nil
	}

	img := doc.Images[sourceIndex]
	if img == nil {
		return nil, nil
	}

	var imagePath string

	// glTF может хранить изображения как URI или как буферы
	if img.URI != "" {
		if gltf.IsDataURI(img.URI) {
			// Data URI - встроенное изображение
			// Пока пропустим, вернем nil
			return nil, nil
		} else {
			// Внешний файл
			imagePath = filepath.Join(baseDir, filepath.FromSlash(img.URI))
		}
	} else if img.BufferView != nil {
		// Изображение в буфере (GLB)
		// Пока не поддерживаем, вернем nil
		return nil, nil
	} else {
		return nil, nil
	}

	if imagePath != "" {
		return LoadTextureFromFile(imagePath)
	}

	return nil, nil
}