//go:build ignore

package main

import (
	"fmt"
	"math"
	"time"

	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/plugin"
)

// ExamplePlugin пример плагина, который вращает все объекты с тегом "rotating"
type ExamplePlugin struct {
	name        string
	version     string
	description string
}

// Plugin экспортируемый символ плагина
var Plugin ExamplePlugin

func init() {
	Plugin = ExamplePlugin{
		name:        "ExamplePlugin",
		version:     "1.0.0",
		description: "Example plugin that rotates objects",
	}
}

func (p ExamplePlugin) Init(manager *plugin.Manager) error {
	fmt.Printf("ExamplePlugin %s initialized\n", p.version)
	return nil
}

func (p ExamplePlugin) Update(deltaTime float32) error {
	// Плагин может выполнять свою логику здесь
	return nil
}

func (p ExamplePlugin) Shutdown() error {
	fmt.Println("ExamplePlugin shut down")
	return nil
}

func (p ExamplePlugin) Name() string {
	return p.name
}

func (p ExamplePlugin) Version() string {
	return p.version
}

func (p ExamplePlugin) Description() string {
	return p.description
}

func (p ExamplePlugin) CreateSystem(world *ecs.World) plugin.System {
	return &RotationSystem{}
}

// RotationSystem система, которая вращает объекты
type RotationSystem struct {
	startTime time.Time
}

func (rs *RotationSystem) Init(world *ecs.World) error {
	rs.startTime = time.Now()
	fmt.Println("RotationSystem initialized")
	return nil
}

func (rs *RotationSystem) Update(world *ecs.World, deltaTime float32) error {
	elapsed := time.Since(rs.startTime).Seconds()

	// Вращаем все объекты с компонентом Transform
	for _, entityID := range world.Entities() {
		if transform, ok := world.GetTransform(entityID); ok {
			// Добавляем вращение вокруг Y оси
			rotationSpeed := float32(30.0) // градусов в секунду
			transform.Rotation.Y += rotationSpeed * deltaTime

			// Добавляем небольшое вертикальное движение
			transform.Position.Y = float32(2.0 + math.Sin(elapsed*2.0)*0.5)

			// Обновляем трансформ
			transform.Position.X = float32(math.Sin(elapsed) * 2.0)
			transform.Position.Z = float32(math.Cos(elapsed) * 2.0)

			world.SetTransform(entityID, transform)
		}
	}

	return nil
}

func (rs *RotationSystem) Shutdown() error {
	fmt.Println("RotationSystem shut down")
	return nil
}

// Создаем дополнительную систему для примера
type DebugSystem struct{}

func (ds *DebugSystem) Init(world *ecs.World) error {
	fmt.Println("DebugSystem initialized")
	return nil
}

func (ds *DebugSystem) Update(world *ecs.World, deltaTime float32) error {
	// Каждые 5 секунд выводим информацию об объектах
	staticCounter := 0
	staticCounter++
	if staticCounter%300 == 0 { // Примерно каждые 5 секунд при 60 FPS
		fmt.Printf("DebugSystem: %d entities in world\n", len(world.Entities()))

		for _, entityID := range world.Entities() {
			if transform, ok := world.GetTransform(entityID); ok {
				fmt.Printf("  Entity %d: pos=(%.2f,%.2f,%.2f)\n",
					entityID, transform.Position.X, transform.Position.Y, transform.Position.Z)
			}
		}
	}

	return nil
}

func (ds *DebugSystem) Shutdown() error {
	fmt.Println("DebugSystem shut down")
	return nil
}