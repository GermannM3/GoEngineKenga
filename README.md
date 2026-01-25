# GoEngineKenga

Полностью автономный игровой движок на Go. Не требует платных сервисов или внешних зависимостей (кроме стандартных Go-библиотек).

## Возможности

### Ядро движка
- **ECS**: Entity-Component-System архитектура
- **Компоненты**: Transform, Camera, MeshRenderer, Rigidbody, Collider, Light, AudioSource, UICanvas
- **Сцены и префабы**: JSON формат
- **Физика**: гравитация, столкновения объект-объект (AABB, Sphere), импульсы
- **Ввод**: клавиатура, мышь (IsKeyPressed, IsMouseButtonPressed, etc.)
- **Аудио**: WAV/MP3/OGG, 3D spatial audio
- **UI**: Button, Label, Panel с автоматическим hover/click

### Графика
- **Ebiten**: 2D рендер, wireframe glTF, splash screen с логотипом
- **glTF 2.0**: импорт моделей → меши и материалы
- **WebGPU**: заглушка для будущего 3D

### Разработка
- **CLI**: `kenga new`, `kenga run`, `kenga import`, `kenga script build`
- **WASM-скриптинг**: TinyGo → WASM, горячая перезагрузка
- **Asset pipeline**: автоматический импорт в `.kenga/`
- **Плагины**: расширяемая архитектура

## Быстрый старт

### Требования
- Go 1.22+
- TinyGo (опционально, для WASM скриптов)

### Установка

```bash
git clone https://github.com/GermannM3/GoEngineKenga.git
cd GoEngineKenga
go mod tidy
```

### Создание нового проекта

```bash
# Создать проект (шаблоны: default, platformer, topdown)
go run ./cmd/kenga new mygame --template platformer
cd mygame

# Запустить игру
go run ../cmd/kenga run --project . --scene scenes/main.scene.json --backend ebiten
```

### Запуск примера

```bash
# Импорт ассетов
go run ./cmd/kenga import --project samples/hello

# Запуск
go run ./cmd/kenga run --project samples/hello --scene scenes/main.scene.json --backend ebiten
```

## Структура проекта игры

```
mygame/
├── project.kenga.json      # Конфигурация проекта
├── scenes/
│   └── main.scene.json     # Сцена
├── assets/                  # Ресурсы (модели, текстуры, звуки)
└── scripts/
    └── game/
        └── main.go         # WASM скрипты
```

## Пример сцены (JSON)

```json
{
  "name": "Main Scene",
  "entities": [
    {
      "name": "Player",
      "components": {
        "transform": {
          "position": {"x": 0, "y": 2, "z": 0},
          "scale": {"x": 1, "y": 1, "z": 1}
        },
        "rigidbody": {
          "mass": 1.0,
          "useGravity": true
        },
        "collider": {
          "type": "box",
          "size": {"x": 1, "y": 2, "z": 1}
        }
      }
    }
  ]
}
```

## Скриптинг (WASM)

```go
//go:build wasm

package main

import "unsafe"

//go:wasmimport env debugLog
func debugLog(ptr uint32, l uint32)

//go:wasmimport env getInputKey
func getInputKey(key int32) int32

const KeyW, KeyA, KeyS, KeyD = 22, 0, 18, 3

//export Update
func Update(dtMillis int32) {
    if getInputKey(KeyW) != 0 {
        // Двигаться вперёд
    }
}

func main() {}
```

Сборка скриптов:
```bash
go run ./cmd/kenga script build --project .
```

## Компоненты ECS

| Компонент | Описание |
|-----------|----------|
| Transform | Позиция, поворот, масштаб |
| Camera | Камера (FOV, near, far) |
| MeshRenderer | Отрисовка меша |
| Light | Освещение (directional, point) |
| Rigidbody | Физическое тело (масса, гравитация) |
| Collider | Коллайдер (box, sphere, capsule) |
| AudioSource | Источник звука (3D spatial) |
| UICanvas | UI холст |

## CLI Команды

| Команда | Описание |
|---------|----------|
| `kenga new <name>` | Создать проект |
| `kenga run --project <path>` | Запустить игру |
| `kenga import --project <path>` | Импортировать ассеты |
| `kenga script build --project <path>` | Собрать WASM скрипты |

## Структура движка

```
engine/
├── ecs/        # Entity-Component-System
├── render/     # Графика (Ebiten, WebGPU)
├── physics/    # Физика и коллизии
├── audio/      # Аудиосистема
├── ui/         # UI элементы
├── input/      # Ввод (клавиатура, мышь)
├── scene/      # Загрузка сцен
├── asset/      # Asset pipeline
├── script/     # WASM runtime
└── cli/        # CLI инструменты
```

## Без платных зависимостей

Движок полностью автономный:
- **Ebiten** — бесплатный, MIT лицензия
- **glTF** — открытый формат моделей
- **Все остальное** — написано с нуля

Никаких облачных сервисов, подписок или скрытых платежей.

## Вклад в проект

Pull requests приветствуются!

## Лицензия

MIT

---

[GitHub](https://github.com/GermannM3/GoEngineKenga)
