package render

import (
	"image/color"

	"goenginekenga/engine/ecs"
)

type Frame struct {
	ClearColor color.RGBA
	World      *ecs.World

	// ProjectDir нужен для резолва ассетов в рантайме (v0).
	ProjectDir string

	// OnUpdate вызывается каждый кадр (dt в секундах). Нужен для единого game loop в любом backend.
	OnUpdate func(dt float64)
}

type Backend interface {
	RunLoop(initial *Frame) error
}
