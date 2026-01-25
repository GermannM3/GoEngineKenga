package ebiten

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
)

type camera struct {
	pos emath.Vec3
	fovY float64
	near float64
	far  float64
}

func defaultCamera() camera {
	return camera{
		pos:  emath.V3(0, 0, 5),
		fovY: 60,
		near: 0.1,
		far:  1000,
	}
}

func findCamera(w *ecs.World) camera {
	cam := defaultCamera()
	for _, id := range w.Entities() {
		if c, ok := w.GetCamera(id); ok {
			if t, ok2 := w.GetTransform(id); ok2 {
				cam.pos = t.Position
			}
			if c.FovYDegrees > 0 {
				cam.fovY = float64(c.FovYDegrees)
			}
			if c.Near > 0 {
				cam.near = float64(c.Near)
			}
			if c.Far > 0 {
				cam.far = float64(c.Far)
			}
			break
		}
	}
	return cam
}

func drawWireframe(screen *ebiten.Image, w *ecs.World, resolver *asset.Resolver, logf func(format string, args ...any)) {
	lineColor := color.RGBA{R: 255, G: 0, B: 0, A: 255} // Красный для отладки

	sw, sh := screen.Size()

	for _, id := range w.Entities() {
		mr, ok := w.GetMeshRenderer(id)
		if !ok || mr.MeshAssetID == "" || resolver == nil {
			logf("entity %d: no mesh renderer or asset id empty\n", id)
			continue
		}

		logf("entity %d: trying to resolve mesh %s\n", id, mr.MeshAssetID)
		mesh, err := resolver.ResolveMeshByAssetID(mr.MeshAssetID)
		if err != nil {
			logf("entity %d: failed to resolve mesh: %v\n", id, err)
			continue
		}
		if mesh == nil {
			logf("entity %d: mesh is nil\n", id)
			continue
		}

		logf("entity %d: mesh loaded, positions: %d, indices: %d\n", id, len(mesh.Positions)/3, len(mesh.Indices))
		tr, _ := w.GetTransform(id)
		logf("entity %d: transform pos=(%.2f,%.2f,%.2f) scale=(%.2f,%.2f,%.2f)\n",
			id, tr.Position.X, tr.Position.Y, tr.Position.Z, tr.Scale.X, tr.Scale.Y, tr.Scale.Z)

		// Простая 2D проекция без камеры - просто используем X,Y и игнорируем Z
		getPos2D := func(vi uint32) (float64, float64, bool) {
			base := int(vi) * 3
			if base+1 >= len(mesh.Positions) {
				return 0, 0, false
			}
			x := float64(mesh.Positions[base] * tr.Scale.X + tr.Position.X)
			y := float64(mesh.Positions[base+1] * tr.Scale.Y + tr.Position.Y)

			// Преобразуем в экранные координаты (центр экрана = 0,0)
			sx := (x * 50) + float64(sw)/2  // Масштаб 50 пикселей на единицу
			sy := (-y * 50) + float64(sh)/2 // Y инвертирован
			return sx, sy, true
		}

		for i := 0; i+2 < len(mesh.Indices); i += 3 {
			i0, i1, i2 := mesh.Indices[i], mesh.Indices[i+1], mesh.Indices[i+2]
			x0, y0, ok0 := getPos2D(i0)
			x1, y1, ok1 := getPos2D(i1)
			x2, y2, ok2 := getPos2D(i2)
			if !(ok0 && ok1 && ok2) {
				continue
			}
			if ok0 && ok1 {
				ebitenutil.DrawLine(screen, x0, y0, x1, y1, lineColor)
			}
			if ok1 && ok2 {
				ebitenutil.DrawLine(screen, x1, y1, x2, y2, lineColor)
			}
			if ok2 && ok0 {
				ebitenutil.DrawLine(screen, x2, y2, x0, y0, lineColor)
			}
		}
	}
}

