//go:build android || ios

package mobile

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/mobile"

	"goenginekenga/engine/render"
	"goenginekenga/engine/render/ebiten"
	"goenginekenga/engine/runtime"
	"goenginekenga/engine/scene"
)

func init() {
	s := scene.DefaultScene()
	rt := runtime.NewFromScene(s)
	rt.StartPlay()

	world, _ := rt.ActiveWorld()

	frame := &render.Frame{
		ClearColor: color.RGBA{R: 15, G: 18, B: 24, A: 255},
		World:      world,
	}
	frame.OnUpdate = func(dt float64) {
		delta := rt.Step()
		if aw, err := rt.ActiveWorld(); err == nil {
			runtime.SpinSystem(aw, delta)
			frame.World = aw
		}
	}

	backend := ebiten.New("GoEngineKenga", 320, 240)
	backend.SetFrame(frame)
	mobile.SetGame(backend)
}

// Dummy — экспортируемая заглушка для gomobile (пакет без экспорта не компилируется).
func Dummy() {}
