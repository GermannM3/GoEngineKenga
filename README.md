# GoEngineKenga

Игровой движок на Go. ECS, сцены, физика, рендер (Ebiten), редактор, WASM-скриптинг.

## Возможности

### Ядро движка
- ECS: сущности, компоненты (Transform, Camera, MeshRenderer, Rigidbody, Collider и др.)
- Сцены и префабы (JSON)
- Физика: rigidbody, коллайдеры, гравитация, отскок от пола
- Аудио, UI: заглушки под будущее

### Графика
- Рендер через Ebiten: wireframe glTF, тестовый треугольник
- glTF 2.0 импорт → меши
- WebGPU: заглушка

### Разработка
- Редактор: Hierarchy, Inspector, Content, Play/Stop, Save, импорт
- WASM-скриптинг (TinyGo), горячая перезагрузка
- Asset pipeline: импорт в `.kenga/`
- Плагины: интерфейс есть, пример в `examples/plugins`

### Инструменты
- Профилировщик, отладчик: заглушки
- CLI: `import`, `run`, `script build`

## Быстрый старт

### Требования
- Go 1.22+
- TinyGo (если нужен WASM)

### Установка и запуск

1. Клонировать репозиторий:
   ```bash
   git clone https://github.com/GermannM3/GoEngineKenga.git
   cd GoEngineKenga
   ```

2. Установить зависимости: `go mod tidy`

3. Запустить пример:
   ```bash
   # Импорт ассетов
   go run ./cmd/kenga import --project samples/hello

   # Запуск игры
   go run ./cmd/kenga run --project samples/hello --scene scenes/main.scene.json --backend ebiten

   # Запуск редактора
   go run ./cmd/kenga-editor
   ```

Своя игра: [CREATING_A_GAME.md](CREATING_A_GAME.md) — как по ссылке на репо начать, что работает, чего нет.

## Структура проекта

```
GoEngineKenga/
├── cmd/                    # Исполняемые файлы
│   ├── kenga/             # CLI инструмент
│   └── kenga-editor/      # Редактор
├── engine/                 # Ядро движка
│   ├── ecs/               # Entity-Component-System
│   ├── render/            # Графическая подсистема
│   ├── physics/           # Физический движок
│   ├── audio/             # Аудиосистема
│   ├── ui/                # UI система
│   ├── scene/             # Сцены и префабы
│   ├── asset/             # Управление ресурсами
│   ├── script/            # WASM скриптинг
│   ├── plugin/            # Система плагинов
│   ├── profiler/          # Профилирование
│   └── debug/             # Отладка
├── editor/                 # Визуальный редактор
├── samples/                # Примеры проектов
└── examples/               # Примеры кода
    └── plugins/           # Примеры плагинов
```

## Компоненты ECS

- Transform, Camera, MeshRenderer, Light
- Rigidbody, Collider (box, sphere, capsule)
- AudioSource, UICanvas

## Скриптинг

TinyGo компилирует Go в WASM:

```go
// scripts/game/main.go
package main

import "unsafe"

//go:wasmimport env debugLog
func debugLog(ptr uint32, l uint32)

func debug(msg string) {
    ptr := uintptr(unsafe.Pointer(&[]byte(msg)[0]))
    debugLog(uint32(ptr), uint32(len(msg)))
}

//export Update
func Update(dtMillis int32) {
    debug("Hello from WASM!")
}
```

Сборка:
```bash
go run ./cmd/kenga script build --project samples/hello
```

## Плагины

Интерфейс `Plugin`: Init, Update, CreateSystem. Пример в `examples/plugins`.

## Пример сцены с физикой

См. `samples/hello/scenes/physics_test.scene.json`. Сущности с `transform`, `rigidbody`, `collider`. Камера с `camera`.

## Сравнение с Unity

| Функция | Unity | GoEngineKenga |
|---------|-------|---------------|
| Язык | C# | Go |
| ECS | ✅ | ✅ |
| Физика | ✅ | ✅ |
| Аудио | ✅ | ✅ |
| UI | ✅ | ✅ |
| Скриптинг | C# | WASM (Go) |
| Плагины | ✅ | ✅ |
| Кроссплатформа | ✅ | ✅ |
| Размер | Большой | Маленький |
| Скорость сборки | Средняя | Быстрая |

## Вклад

Pull requests приветствуются. Планы: WebGPU/3D, коллизии объект–объект, ввод, аудио, UI.

## Лицензия

MIT.

[https://github.com/GermannM3/GoEngineKenga](https://github.com/GermannM3/GoEngineKenga)
