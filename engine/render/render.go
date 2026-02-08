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

	// Resolver — если задан, backend использует его для загрузки мешей/материалов.
	// interface{} для избежания цикла импортов; backend делает type assertion на *asset.Resolver.
	Resolver interface{}

	// OnUpdate вызывается каждый кадр (dt в секундах). Нужен для единого game loop в любом backend.
	OnUpdate func(dt float64)

	// OnFrameRendered вызывается после отрисовки кадра (screen — *ebiten.Image или аналог).
	// Используется для стриминга viewport в IDE по WebSocket.
	OnFrameRendered func(screen interface{})

	// OrbitResetRequested — при true backend сбрасывает orbit camera и синхронизирует с камерой сцены (hot-reload).
	OrbitResetRequested bool

	// InvalidateMeshCache — при true WebGPU backend очищает кэш мешей (hot-reload ассетов).
	InvalidateMeshCache bool
}

type Backend interface {
	RunLoop(initial *Frame) error
}
