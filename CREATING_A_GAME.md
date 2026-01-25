# Как создать игру на GoEngineKenga

## 1. Создание проекта

### Способ 1: Команда `kenga new`

```bash
git clone https://github.com/GermannM3/GoEngineKenga.git
cd GoEngineKenga
go mod tidy

# Создать проект с шаблоном
go run ./cmd/kenga new mygame --template platformer
cd mygame

# Запустить
go run ../cmd/kenga run --project . --scene scenes/main.scene.json --backend ebiten
```

Доступные шаблоны:
- `default` — пустой проект с кубом
- `platformer` — игрок с гравитацией и платформой
- `topdown` — игрок без гравитации (вид сверху)

### Способ 2: Вручную

Структура проекта:
```
mygame/
├── project.kenga.json
├── scenes/
│   └── main.scene.json
├── assets/
└── scripts/
    └── game/
        └── main.go
```

`project.kenga.json`:
```json
{
  "name": "mygame",
  "scenes": ["scenes/main.scene.json"],
  "assetsDir": "assets",
  "derivedDir": ".kenga/derived"
}
```

## 2. Что работает

| Функция | Статус | Описание |
|---------|--------|----------|
| ECS | ✅ | Сущности и компоненты |
| Сцены JSON | ✅ | Загрузка и сохранение |
| Transform | ✅ | Позиция, поворот, масштаб |
| Camera | ✅ | Камера с FOV |
| MeshRenderer | ✅ | Wireframe glTF |
| Rigidbody | ✅ | Масса, гравитация, скорость |
| Collider | ✅ | Box, Sphere — коллизии объект-объект |
| Физика | ✅ | Гравитация, импульсы, отскок |
| Ввод | ✅ | Клавиатура, мышь, скролл |
| Аудио | ✅ | WAV/MP3/OGG, 3D spatial |
| UI | ✅ | Button, Label, Panel |
| WASM | ✅ | TinyGo скрипты |
| Asset Import | ✅ | glTF → меши |
| Редактор | ⚠️ | Требует CGO/Fyne |
| 3D рендер | ❌ | Только wireframe |
| WebGPU | ❌ | Заглушка |

## 3. Пример сцены с физикой

```json
{
  "name": "Physics Demo",
  "entities": [
    {
      "name": "Camera",
      "components": {
        "transform": {
          "position": {"x": 0, "y": 5, "z": 10}
        },
        "camera": {
          "fovYDegrees": 60,
          "near": 0.1,
          "far": 1000
        }
      }
    },
    {
      "name": "Ball",
      "components": {
        "transform": {
          "position": {"x": 0, "y": 5, "z": 0},
          "scale": {"x": 1, "y": 1, "z": 1}
        },
        "rigidbody": {
          "mass": 1.0,
          "useGravity": true
        },
        "collider": {
          "type": "sphere",
          "radius": 0.5
        }
      }
    },
    {
      "name": "Ground",
      "components": {
        "transform": {
          "position": {"x": 0, "y": 0, "z": 0},
          "scale": {"x": 10, "y": 0.5, "z": 10}
        },
        "collider": {
          "type": "box",
          "size": {"x": 10, "y": 0.5, "z": 10}
        }
      }
    }
  ]
}
```

## 4. Скрипты (WASM)

`scripts/game/main.go`:
```go
//go:build wasm

package main

import "unsafe"

//go:wasmimport env debugLog
func debugLog(ptr uint32, l uint32)

//go:wasmimport env getInputKey
func getInputKey(key int32) int32

func log(msg string) {
    if len(msg) == 0 { return }
    b := []byte(msg)
    debugLog(uint32(uintptr(unsafe.Pointer(&b[0]))), uint32(len(b)))
}

// Клавиши
const (
    KeyW = 22
    KeyA = 0
    KeyS = 18
    KeyD = 3
    KeySpace = 36
)

//export Update
func Update(dtMillis int32) {
    if getInputKey(KeyW) != 0 {
        log("W pressed\n")
    }
    if getInputKey(KeySpace) != 0 {
        log("Jump!\n")
    }
}

func main() {}
```

Сборка:
```bash
go run ./cmd/kenga script build --project .
```

## 5. Команды CLI

```bash
# Создать проект
go run ./cmd/kenga new mygame

# Импортировать ассеты
go run ./cmd/kenga import --project mygame

# Запустить игру
go run ./cmd/kenga run --project mygame --scene scenes/main.scene.json --backend ebiten

# Собрать WASM скрипты
go run ./cmd/kenga script build --project mygame
```

## 6. Системные требования

- Go 1.22+
- TinyGo (опционально, для WASM)
- Никаких платных зависимостей
