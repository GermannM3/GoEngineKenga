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
}

type Backend interface {
	RunLoop(initial *Frame) error
}
