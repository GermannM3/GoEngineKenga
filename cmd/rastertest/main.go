// Изолированная проверка растеризатора: рисует один куб через камер
// как делает Renderer3D, и считает непустые пиксели. Не часть релиза.
package main

import (
	"fmt"
	"image/color"

	emath "goenginekenga/engine/math"
	"goenginekenga/engine/render"
)

func main() {
	const W, H = 320, 180
	ras := render.NewRasterizer(W, H)
	cam := render.NewCamera3D()
	cam.SetPosition(emath.Vec3{X: 0, Y: 2, Z: 8})
	cam.SetTarget(emath.Vec3{X: 0, Y: 1, Z: 0})
	cam.SetFOV(60)
	cam.SetAspectRatio(float32(W) / float32(H))
	cam.GetViewProjectionMatrix() // предвычисление
	ras.SetCamera(cam)

	ras.Clear(color.RGBA{R: 20, G: 20, B: 30, A: 255})
	ras.SetAmbientColor(color.RGBA{R: 120, G: 120, B: 120, A: 255})
	ras.AddLight(render.Light3D{
		Type:      "point",
		Position:  emath.Vec3{X: 5, Y: 8, Z: 2},
		Color:     color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Intensity: 1.2,
		Range:     40,
	})

	cube := render.CreateCube()
	model := render.Translate(emath.Vec3{X: 0, Y: 1, Z: 0})
	ras.DrawMesh(cube.Vertices, cube.Indices, cube.Normals, cube.UVs, model, nil, nil, 1.0, color.RGBA{R: 200, G: 90, B: 60, A: 255})

	img := ras.RenderToImage()
	nonBg := 0
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			r, g, b, _ := img.RGBAAt(x, y).RGBA()
			if r>>8 != 20 || g>>8 != 20 || b>>8 != 30 {
				nonBg++
			}
		}
	}
	fmt.Printf("non-background pixels: %d / %d\n", nonBg, W*H)
}