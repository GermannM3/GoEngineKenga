package render

import (
	"image"
	"image/color"
)

// FrameRenderer defines the interface for renderers that can output frame data
type FrameRenderer interface {
	RenderToImage() *image.RGBA
}

// RenderToBuffer возвращает текущий цветовой буфер как байтовый массив в формате RGBA.
// Возвращает байты, ширину, высоту и ошибку.
func (r *Rasterizer) RenderToBuffer() ([]byte, int, int, error) {
	img := r.RenderToImage()
	if img == nil {
		return nil, 0, 0, nil
	}

	// Преобразуем изображение в байтовый массив
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	
	// Создаем байтовый слайс для RGBA данных
	buffer := make([]byte, width*height*4)
	
	// Копируем данные изображения в буфер
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.RGBAAt(x, y)
			i := (y*width + x) * 4
			buffer[i] = c.R
			buffer[i+1] = c.G
			buffer[i+2] = c.B
			buffer[i+3] = c.A
		}
	}

	return buffer, width, height, nil
}

// RenderToImageWithFormat возвращает изображение с заданным форматом
func (r *Rasterizer) RenderToImageWithFormat(targetImg *image.RGBA) error {
	sourceImg := r.RenderToImage()
	if sourceImg == nil || targetImg == nil {
		return nil
	}

	// Копируем пиксели из одного изображения в другое
	bounds := sourceImg.Bounds()
	if bounds.Dx() != targetImg.Bounds().Dx() || bounds.Dy() != targetImg.Bounds().Dy() {
		// Изменяем размер целевого изображения, если нужно
		return nil
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := sourceImg.RGBAAt(x, y)
			targetImg.SetRGBA(x, y, c)
		}
	}

	return nil
}

// GetBufferSize возвращает размер буфера, необходимый для хранения изображения заданного размера
func GetBufferSize(width, height int) int {
	return width * height * 4 // 4 bytes per pixel (RGBA)
}

// ConvertRGBABytesToImage преобразует байтовый массив RGBA в *image.RGBA
func ConvertRGBABytesToImage(data []byte, width, height int) *image.RGBA {
	if len(data) != width*height*4 {
		return nil
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := (y*width + x) * 4
			img.SetRGBA(x, y, color.RGBA{
				R: data[i],
				G: data[i+1],
				B: data[i+2],
				A: data[i+3],
			})
		}
	}

	return img
}