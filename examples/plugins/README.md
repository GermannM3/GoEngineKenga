# Плагины GoEngineKenga

Расширение движка без правок в ядре. Интерфейс `Plugin`: Init, Update, Shutdown, Name, Version, Description, CreateSystem.

## Создание плагина

1. Новый пакет Go, реализовать `Plugin`
2. Экспортировать переменную `Plugin`
3. Сборка: `go build -buildmode=plugin -o plugin.so`

## Пример плагина

```go
package main

import (
    "goenginekenga/engine/ecs"
    "goenginekenga/engine/plugin"
)

type MyPlugin struct {
    name string
}

var Plugin = MyPlugin{name: "MyPlugin"}

func (p MyPlugin) Init(manager *plugin.Manager) error {
    // Инициализация
    return nil
}

func (p MyPlugin) Update(deltaTime float32) error {
    // Обновление каждый кадр
    return nil
}

func (p MyPlugin) Shutdown() error {
    // Завершение
    return nil
}

func (p MyPlugin) Name() string { return p.name }
func (p MyPlugin) Version() string { return "1.0.0" }
func (p MyPlugin) Description() string { return "Example plugin" }

// Если плагин добавляет систему ECS:
func (p MyPlugin) CreateSystem(world *ecs.World) plugin.System {
    return &MySystem{}
}

type MySystem struct{}

func (s *MySystem) Init(world *ecs.World) error { return nil }
func (s *MySystem) Update(world *ecs.World, deltaTime float32) error { return nil }
func (s *MySystem) Shutdown() error { return nil }
```

## Сборка плагина

```bash
go build -buildmode=plugin -o myplugin.so myplugin.go
```

## Использование

```go
manager := plugin.NewManager(runtime)
_ = manager.LoadPlugin("./plugins/myplugin.so")
// В цикле: manager.Update(deltaTime); manager.Render(world)
```

API: Init, Update, Shutdown, Name, Version, Description. Менеджер: LoadPlugin, UnloadPlugin, Update, Render, ListPlugins. Подробнее — `engine/plugin/plugin.go`, пример — `example_plugin.go`.