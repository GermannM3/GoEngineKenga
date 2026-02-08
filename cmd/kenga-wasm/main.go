//go:build js

// kenga-wasm — точка входа для сборки в WebAssembly.
// Запускает 3D-viewport с дефолтной сценой (куб) в браузере.
//
// Сборка:
//
//	go run github.com/hajimehoshi/wasmserve@latest ./cmd/kenga-wasm
//
// Или вручную:
//
//	GOOS=js GOARCH=wasm go build -o kenga.wasm ./cmd/kenga-wasm
//	cp $(go env GOROOT)/lib/wasm/wasm_exec.js .
package main

import (
	"image/color"

	"goenginekenga/engine/render"
	"goenginekenga/engine/render/ebiten"
	"goenginekenga/engine/runtime"
	"goenginekenga/engine/scene"
)

func main() {
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

	backend := ebiten.New("GoEngineKenga", 1280, 720)
	_ = backend.RunLoop(frame)
}
