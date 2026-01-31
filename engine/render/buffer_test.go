package render

import (
	"image"
	"image/color"
	"testing"
)

func TestRenderToBuffer(t *testing.T) {
	// Создаем растеризатор
	r := NewRasterizer(100, 100)

	// Очищаем буфер
	clearColor := color.RGBA{R: 255, G: 0, B: 0, A: 255} // Красный
	r.Clear(clearColor)

	// Вызываем RenderToBuffer
	buffer, width, height, err := r.RenderToBuffer()
	if err != nil {
		t.Fatalf("RenderToBuffer returned error: %v", err)
	}

	if width != 100 {
		t.Errorf("Expected width 100, got %d", width)
	}

	if height != 100 {
		t.Errorf("Expected height 100, got %d", height)
	}

	expectedSize := 100 * 100 * 4 // 4 байта на пиксель (RGBA)
	if len(buffer) != expectedSize {
		t.Errorf("Expected buffer size %d, got %d", expectedSize, len(buffer))
	}

	// Проверяем, что первый пиксель красный
	if buffer[0] != 255 || buffer[1] != 0 || buffer[2] != 0 || buffer[3] != 255 {
		t.Errorf("Expected red pixel [255, 0, 0, 255], got [%d, %d, %d, %d]", 
			buffer[0], buffer[1], buffer[2], buffer[3])
	}
}

func TestConvertRGBABytesToImage(t *testing.T) {
	// Создаем тестовые байты для изображения 2x2
	testData := []byte{
		255, 0, 0, 255,     // Красный пиксель
		0, 255, 0, 255,     // Зеленый пиксель
		0, 0, 255, 255,     // Синий пиксель
		255, 255, 255, 255, // Белый пиксель
	}

	img := ConvertRGBABytesToImage(testData, 2, 2)
	if img == nil {
		t.Fatal("ConvertRGBABytesToImage returned nil")
	}

	// Проверяем размеры
	bounds := img.Bounds()
	if bounds.Dx() != 2 || bounds.Dy() != 2 {
		t.Errorf("Expected image size 2x2, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Проверяем пиксели
	expectedColors := []color.RGBA{
		{R: 255, G: 0, B: 0, A: 255},   // (0,0) - красный
		{R: 0, G: 255, B: 0, A: 255},   // (1,0) - зеленый
		{R: 0, G: 0, B: 255, A: 255},   // (0,1) - синий
		{R: 255, G: 255, B: 255, A: 255}, // (1,1) - белый
	}

	y := 0
	x := 0
	for i, expectedColor := range expectedColors {
		actualColor := img.RGBAAt(x, y)
		if actualColor != expectedColor {
			t.Errorf("Pixel at (%d,%d) - Expected %v, got %v", x, y, expectedColor, actualColor)
		}

		x++
		if x == 2 {
			x = 0
			y++
		}
	}
}

func TestGetBufferSize(t *testing.T) {
	width := 100
	height := 50
	expected := width * height * 4 // 4 байта на пиксель
	actual := GetBufferSize(width, height)

	if actual != expected {
		t.Errorf("For size %dx%d, expected buffer size %d, got %d", width, height, expected, actual)
	}
}

func TestRasterizerImplementsFrameRenderer(t *testing.T) {
	r := NewRasterizer(10, 10)
	
	// Проверяем, что растеризатор реализует интерфейс FrameRenderer
	var fr FrameRenderer = r
	
	// Проверяем, что оба метода работают
	img := fr.RenderToImage()
	if img == nil {
		t.Error("RenderToImage returned nil")
	}
	
	buf, w, h, err := fr.RenderToBuffer()
	if err != nil {
		t.Errorf("RenderToBuffer returned error: %v", err)
	}
	if buf == nil {
		t.Error("RenderToBuffer returned nil buffer")
	}
	if w != 10 || h != 10 {
		t.Errorf("RenderToBuffer returned wrong dimensions: %dx%d", w, h)
	}
}