//go:build ignore

package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

func main() {
	root, _ := os.Getwd()
	// если запуск из scripts/
	if filepath.Base(root) == "scripts" {
		root = filepath.Dir(root)
	}
	base := filepath.Join(root, "assets")
	cars := filepath.Join(base, "cars")
	_ = os.MkdirAll(cars, 0755)

	writeCar(filepath.Join(cars, "atom.png"), color.RGBA{0, 200, 255, 255})
	writeCar(filepath.Join(cars, "moskvich_m70.png"), color.RGBA{255, 120, 40, 255})
	writeCar(filepath.Join(cars, "moskvich_m90.png"), color.RGBA{60, 80, 255, 255})
	writeBox(filepath.Join(base, "powerup_box.png"), color.RGBA{255, 220, 40, 255})
	writeTrack(filepath.Join(base, "track_placeholder.png"))
}

func writeCar(path string, body color.RGBA) {
	w, h := 64, 36
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}
	// корпус
	for y := 8; y < 28; y++ {
		for x := 6; x < 58; x++ {
			m.Set(x, y, body)
		}
	}
	// кабина
	cab := color.RGBA{uint8(float32(body.R) * 0.45), uint8(float32(body.G) * 0.45), uint8(float32(body.B) * 0.55), 255}
	for y := 10; y < 20; y++ {
		for x := 22; x < 48; x++ {
			m.Set(x, y, cab)
		}
	}
	// колёса
	wheel := color.RGBA{25, 25, 30, 255}
	for y := 24; y < 34; y++ {
		for x := 4; x < 14; x++ {
			m.Set(x, y, wheel)
		}
		for x := 50; x < 60; x++ {
			m.Set(x, y, wheel)
		}
	}
	for y := 4; y < 12; y++ {
		for x := 8; x < 16; x++ {
			m.Set(x, y, wheel)
		}
		for x := 48; x < 56; x++ {
			m.Set(x, y, wheel)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	_ = png.Encode(f, m)
}

func writeBox(path string, c color.RGBA) {
	m := image.NewRGBA(image.Rect(0, 0, 24, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			col := c
			if x < 2 || y < 2 || x > 21 || y > 21 {
				col = color.RGBA{40, 40, 20, 255}
			}
			m.Set(x, y, col)
		}
	}
	f, _ := os.Create(path)
	defer f.Close()
	_ = png.Encode(f, m)
}

func writeTrack(path string) {
	m := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			m.Set(x, y, color.RGBA{70, 75, 85, 255})
		}
	}
	f, _ := os.Create(path)
	defer f.Close()
	_ = png.Encode(f, m)
}
