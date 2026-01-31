# GoEngineKenga

Полнофункциональный игровой движок на Go. Полностью автономный — не требует платных сервисов или внешних зависимостей.

## Возможности

### Ядро движка
- **ECS**: Entity-Component-System архитектура
- **Компоненты**: Transform, Camera, MeshRenderer, Rigidbody, Collider, Light, AudioSource, UICanvas
- **Сцены и префабы**: JSON формат
- **Физика**: гравитация, столкновения объект-объект (AABB, Sphere, Box-Sphere), импульсы
- **Ввод**: клавиатура, мышь (IsKeyPressed, IsMouseButtonPressed, delta, scroll)
- **Аудио**: WAV/MP3/OGG, 3D spatial audio, FFT анализ
- **UI**: Button, Label, Panel с автоматическим hover/click

### 3D Графика
- **Software 3D рендер**: растеризация с Z-буфером
- **Текстуры**: поддержка текстурированных моделей
- **Освещение**: Ambient, Directional, Point lights
- **Камера**: перспектива, LookAt, FOV
- **Меши**: куб, сфера, плоскость, цилиндр + импорт glTF

### Система частиц
- **Эмиттеры**: rate, burst, направление, разброс
- **Физика частиц**: гравитация, drag, скорость
- **Lifetime**: размер/цвет по времени жизни
- **Пресеты**: огонь, дым, взрыв, брызги воды, искры

### Анимация
- **Скелетная**: кости, IK, skinning
- **Keyframe**: позиция, поворот, масштаб
- **Клипы**: loop, speed, crossfade
- **Sprite**: покадровая анимация для 2D

### Процедурная генерация
- **Noise**: Perlin 2D, FBM, Turbulence
- **Heightmap**: генерация ландшафта
- **Dungeon**: BSP-алгоритм, комнаты, коридоры
- **World Map**: острова, архипелаги, биомы

### AI для NPC
- **Pathfinding**: A* на NavGrid
- **Behavior Trees**: Sequence, Selector, Action, Condition
- **State Machine**: состояния, переходы
- **Пресеты**: патруль, преследование

### Физика воды
- **Wave simulation**: динамическая поверхность воды
- **Gerstner waves**: реалистичные океанские волны
- **Buoyancy**: плавучесть объектов
- **Ship physics**: физика корабля с парусом и рулём

### Шейдеры и эффекты
- **Software shaders**: vertex + fragment
- **Toon**: cel-shading с настройкой уровней
- **Psychedelic**: спиральные RGB-эффекты
- **Glitch**: цифровые артефакты
- **Fog**: distance fog
- **Post-process**: vignette, chromatic aberration, outline

### Аудио-реактивность
- **FFT анализ**: spectrum, bands (bass/mid/high)
- **Beat detection**: автоматическое определение ритма
- **Visualizers**: waveform, spectrum bars, circular

### Деформация пространства
- **Wave**: волновая деформация
- **Twist**: скручивание вокруг оси
- **Bend**: изгиб
- **Pulse**: пульсация
- **Noise**: шумовая деформация
- **Sphere attract**: притяжение к сфере
- **Melt**: эффект плавления

### Разработка
- **CLI**: `kenga new`, `kenga run`, `kenga import`, `kenga script build`
- **WASM-скриптинг**: TinyGo → WASM, горячая перезагрузка
- **Asset pipeline**: автоматический импорт в `.kenga/`
- **Плагины**: расширяемая архитектура

## Быстрый старт

### Требования
- Go 1.22+
- TinyGo (опционально, для WASM скриптов)

### Установка и подключение движка

**Вариант 1 — клонировать и собирать локально (для игр и для встраивания в свои проекты):**

```bash
git clone https://github.com/GermannM3/GoEngineKenga.git
cd GoEngineKenga
go mod tidy
go build -o kenga ./cmd/kenga
```

Движок можно вызывать из своих скриптов или приложений (например, CAD-программа на Python запускает `kenga run --project ... --ws-port ...` и управляет сценой по WebSocket). Подробнее: [docs/CAD.md](docs/CAD.md).

**Вариант 2 — установить готовый бинарник:**

- Windows: запустите установщик из [релизов](https://github.com/GermannM3/GoEngineKenga/releases) или соберите один exe: `scripts\make-setup.bat`.
- Linux: `./scripts/install-linux.sh` или пакет .deb: [docs/INSTALLER.md](docs/INSTALLER.md).

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

Если установлен бинарник `kenga` (через релиз/установщик), его можно вызывать напрямую:

```bash
kenga run --project samples/hello --scene scenes/main.scene.json --backend ebiten
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

## Примеры использования

### Частицы (огонь)

```go
import "goenginekenga/engine/particles"

// Создать систему частиц
ps := particles.NewSystem(1000)

// Добавить эмиттер огня
fire := particles.NewFireEmitter(position)
ps.AddEmitter(fire)

// Обновлять каждый кадр
ps.Update(deltaTime)
```

### AI патруль

```go
import "goenginekenga/engine/ai"

// Создать агента
agent := ai.NewAgent(startPos, speed)
agent.NavGrid = navGrid

// Создать поведение патруля
patrol := ai.CreatePatrolBehavior(waypoints)

// Обновлять
patrol.Execute(agent)
agent.Update(dt)
```

### Процедурный dungeon

```go
import "goenginekenga/engine/procgen"

dungeon := procgen.NewDungeon(100, 100)
dungeon.GenerateBSP(seed, minRoomSize, maxRoomSize)

// Получить тайлы
for y := 0; y < dungeon.Height; y++ {
    for x := 0; x < dungeon.Width; x++ {
        tile := dungeon.Get(x, y)
        // TileWall, TileFloor, TileDoor, etc.
    }
}
```

### Океанские волны

```go
import "goenginekenga/engine/physics"

ocean := physics.NewOceanWaves()
ocean.AddWave(direction, wavelength, amplitude, steepness, speed)

// Получить высоту волны
height := ocean.GetHeightAt(x, z)
normal := ocean.GetNormalAt(x, z)

// Обновлять
ocean.Update(dt)
```

### Аудио-реактивность

```go
import "goenginekenga/engine/audio"

analyzer := audio.NewAudioAnalyzer(44100, 2048)
analyzer.PushSamples(audioSamples)
analyzer.Analyze()

// Получить данные
bass := analyzer.Bass
isBeat := analyzer.BeatDetected
spectrum := analyzer.GetSpectrumNormalized()
```

### Деформация меша

```go
import "goenginekenga/engine/render"

deformer := render.NewSpaceDeformer()
deformer.AddDeformation(render.WaveDeform(0.5, 2, 1))
deformer.AddDeformation(render.TwistDeform(0.3))

// Применить к мешу
deformer.DeformMesh(sourceMesh, deformedMesh)
deformer.Update(dt)
```

## Использование как 3D‑движок в CAD/CAM/робототехнике

GoEngineKenga можно использовать не только как игровой движок, но и как **встраиваемый 3D‑визуализатор** для десктоп‑приложений (Python/PyQt, Qt/QML, .NET и т.п.).

### Режим «отдельное окно + WebSocket управление»

Базовый и самый простой способ интеграции — запускать `kenga` как отдельный процесс с собственным окном, а управлять сценой из внешней программы через WebSocket‑API.

#### Запуск рантайма с WebSocket‑сервером

```bash
go run ./cmd/kenga run \
  --project samples/hello \
  --scene scenes/main.scene.json \
  --backend ebiten \
  --ws-port 127.0.0.1:7777
```

- WebSocket‑сервер поднимается по адресу `ws://127.0.0.1:7777/ws`.
- Порт и адрес можно менять флагом `--ws-port`.
- Можно отключить сервер полностью: `--no-ws`.

#### Формат сообщений по WebSocket

Все команды и ответы — JSON‑объекты вида:

```json
{
  "cmd": "set_camera",
  "request_id": "optional-string",
  "data": { ... }
}
```

После приёма команды движок отправляет обратно простой ACK:

```json
{
  "ok": true,
  "cmd": "set_camera",
  "request_id": "optional-string"
}
```

#### Ключевые команды для CAD/робототехники

- **Загрузка модели робота / изделия**:

  ```json
  {
    "cmd": "load_model",
    "data": {
      "asset_id": "ASSET_UUID",
      "path": "assets/robot.gltf",
      "entity_id": "robot1",
      "name": "RobotArm"
    }
  }
  ```

- **Очистка сцены**:

  ```json
  { "cmd": "clear_scene", "data": { "mode": "play" } }
  ```

- **Камера (вид, как в CAD)**:

  ```json
  {
    "cmd": "set_camera",
    "data": {
      "pos": [2.0, 2.0, 5.0],
      "target": [0.0, 0.0, 0.0],
      "fov_deg": 60.0,
      "near": 0.1,
      "far": 1000.0
    }
  }
  ```

- **Положение объектов**:

  ```json
  {
    "cmd": "set_transform",
    "data": {
      "entity_id": "robot1",
      "pos": [x, y, z],
      "rot_deg": [rx, ry, rz],
      "scale": [sx, sy, sz],
      "use_pos": true,
      "use_rot": true,
      "use_scale": true
    }
  }
  ```

- **Траектории движения**:

  - Полная траектория:

    ```json
    {
      "cmd": "set_trajectory",
      "data": {
        "entity_id": "traj1",
        "points": [[x1, y1, z1], [x2, y2, z2]],
        "color_rgba": [255, 200, 80, 255],
        "width": 2.0
      }
    }
    ```

  - Добавить точку:

    ```json
    {
      "cmd": "add_trajectory_point",
      "data": {
        "entity_id": "traj1",
        "point": [x, y, z]
      }
    }
    ```

  - Очистить:

    ```json
    {
      "cmd": "clear_trajectory",
      "data": { "entity_id": "traj1" }
    }
    ```

- **Суставы робота (joint control)**:

  ```json
  {
    "cmd": "set_joint",
    "data": {
      "joint_name": "joint3",
      "angle_deg": 45.0,
      "axis": [0, 1, 0]
    }
  }
  ```

- **Нанесение мастики / клея (dispensing)**:

  - Запуск:

    ```json
    {
      "cmd": "start_dispensing",
      "data": {
        "entity_id": "robot1",
        "flow_rate": 1.5,
        "radius": 0.02,
        "color_rgba": [255, 220, 120, 255]
      }
    }
    ```

  - Остановка:

    ```json
    {
      "cmd": "stop_dispensing",
      "data": { "entity_id": "robot1" }
    }
    ```

### Рекомендуемый паттерн для Python/PyQt

1. Запускать `kenga` как отдельный процесс из Python (через `subprocess.Popen`) с флагами `--project`, `--scene`, `--ws-port`.
2. В PyQt‑приложении держать один постоянный WebSocket‑клиент (например, через `websockets` или `QtWebSockets`), который:
   - после подключения инициализирует камеру, загружает модели робота и стенда;
   - отправляет `set_transform`, `set_joint`, `set_trajectory`, `start_dispensing` при изменении данных в UI;
   - по необходимости запрашивает состояние (будущий метод `get_state`).
3. Вся «умная» логика (кинематика, планирование траекторий, проверка коллизий на уровне сцены) живёт в вашем приложении; движок отвечает за:
   - быструю 3D‑визуализацию,
   - низкоуровневые коллизии,
   - отрисовку траекторий и мастики.

Чтобы явно указать путь к CLI‑бинарнику, можно использовать переменную окружения `KENGA_CLI`:

```bash
set KENGA_CLI=D:\Tools\GoEngineKenga\kenga.exe   # Windows (cmd)
export KENGA_CLI=/opt/goenginekenga/kenga       # Linux/macOS (bash/zsh)
```

Редактор `kenga-editor` также использует `KENGA_CLI`, если она установлена, когда запускает внешний рантайм из тулбара.

### Встраивание рендера через RenderToImage

Софтварный рендер предоставляет доступ к буферу изображения:

- В `engine/render/rasterizer.go`:

  ```go
  func (r *Rasterizer) RenderToImage() *image.RGBA
  ```

- В `engine/render/buffer.go`:

  ```go
  type FrameRenderer interface {
      RenderToImage() *image.RGBA
  }
  ```

- В `engine/render/ebiten/renderer3d.go`:

  ```go
  func (r *Renderer3D) RenderToImage() *image.RGBA
  ```

Это можно обернуть через Cgo в DLL/so/dylib и интегрировать с родным UI‑фреймворком (Qt, wxWidgets и др.), если отдельное окно движка неприменимо. Для начала разработки CAD‑подобного приложения на Python удобно использовать именно WebSocket‑режим, а embed через Cgo оставить как следующий шаг.

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
├── render/     # Графика (3D рендер, камера, меши)
├── physics/    # Физика, коллизии, вода
├── audio/      # Аудио, FFT анализ
├── particles/  # Система частиц
├── animation/  # Анимация (skeletal, sprite)
├── procgen/    # Процедурная генерация
├── ai/         # AI, pathfinding, behavior trees
├── shader/     # Программируемые шейдеры
├── ui/         # UI элементы
├── input/      # Ввод (клавиатура, мышь)
├── scene/      # Загрузка сцен
├── asset/      # Asset pipeline
├── script/     # WASM runtime
└── cli/        # CLI инструменты
```

## Готовность к созданию игр

| Функция | Статус |
|---------|--------|
| 3D рендер с текстурами | ✅ |
| Освещение | ✅ |
| Физика воды | ✅ |
| Система частиц | ✅ |
| Анимация моделей | ✅ |
| Динамическая карта | ✅ |
| AI для NPC | ✅ |
| Пользовательские шейдеры | ✅ |
| Деформация пространства | ✅ |
| Процедурная генерация | ✅ |
| Аудио-реактивные эффекты | ✅ |

## Честная оценка движков

Сравнение GoEngineKenga с Unity и Unreal без маркетинга — что реально есть, где ограничения, когда что выбирать: **[docs/ОЦЕНКА_ДВИЖКОВ.md](docs/ОЦЕНКА_ДВИЖКОВ.md)**.

## Без платных зависимостей

Движок полностью автономный:
- **Ebiten** — бесплатный, MIT лицензия
- **glTF** — открытый формат моделей
- **Всё остальное** — написано с нуля

Никаких облачных сервисов, подписок или скрытых платежей.

## Вклад в проект

Pull requests приветствуются!

## Лицензия

MIT

---

[GitHub](https://github.com/GermannM3/GoEngineKenga)
