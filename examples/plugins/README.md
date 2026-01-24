# GoEngineKenga Plugin System

Система плагинов позволяет расширять функциональность движка без изменения основного кода.

## Создание плагина

1. Создайте новый пакет Go с плагином
2. Реализуйте интерфейс `Plugin`
3. Экспортируйте переменную `Plugin` с экземпляром вашего плагина
4. Соберите плагин как shared library: `go build -buildmode=plugin -o plugin.so`

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
func (p MyPlugin) Description() string { return "My awesome plugin" }

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

## Использование в игре

```go
manager := plugin.NewManager(runtime)
err := manager.LoadPlugin("./plugins/myplugin.so")
if err != nil {
    log.Fatal(err)
}

// В игровом цикле:
manager.Update(deltaTime)
manager.Render(world)
```

## Типы плагинов

- **Plugin**: Базовый плагин
- **SystemPlugin**: Плагин, добавляющий ECS систему
- **RenderPlugin**: Плагин с дополнительным рендерингом

## API плагинов

- `Init(manager *Manager) error` - инициализация
- `Update(deltaTime float32) error` - обновление каждый кадр
- `Shutdown() error` - завершение работы
- `Name() string` - имя плагина
- `Version() string` - версия
- `Description() string` - описание

## Менеджер плагинов

- `LoadPlugin(path string) error` - загрузить плагин
- `UnloadPlugin(name string) error` - выгрузить плагин
- `Update(deltaTime float32)` - обновить все плагины
- `Render(world *ecs.World)` - рендеринг плагинов
- `ListPlugins() []string` - список плагинов