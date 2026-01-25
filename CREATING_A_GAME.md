# Как создать игру на GoEngineKenga

Как по ссылке на репо начать, что уже работает, чего нет, что можно автоматизировать.

## 1. Как по ссылке создать игру

Ссылка: https://github.com/GermannM3/GoEngineKenga

### Шаги

1. Клонировать и подготовить окружение
   ```bash
   git clone https://github.com/GermannM3/GoEngineKenga.git
   cd GoEngineKenga
   go mod tidy
   ```

2. Использовать `samples/hello` или скопировать его структуру в свою папку (`mygame/` и т.п.).

3. Импортировать ассеты (glTF, текстуры из `assets/`)
   ```bash
   go run ./cmd/kenga import --project samples/hello
   ```
   Свой проект: заменить `samples/hello` на путь к папке с `project.kenga.json`.

4. Запустить сцену
   ```bash
   go run ./cmd/kenga run --project samples/hello --scene scenes/main.scene.json --backend ebiten
   ```

5. Редактор (по желанию): `go run ./cmd/kenga-editor`. По умолчанию открывает `samples/hello`. Hierarchy, Inspector, Content, Play/Stop, Save, импорт.

6. WASM (по желанию): [TinyGo](https://tinygo.org/), логика в `scripts/game/main.go`. Сборка: `go run ./cmd/kenga script build --project samples/hello`. При `kenga run` подхватывается `game.wasm` из `.kenga/scripts/`.

### Минимальная структура своего проекта

```
mygame/
├── project.kenga.json     # имя, сцены, assetsDir, derivedDir
├── assets/                # glTF, текстуры и т.д.
├── scenes/
│   └── main.scene.json    # первая сцена
└── scripts/
    └── game/
        └── main.go        # опционально, для WASM
```

Пример `project.kenga.json`:

```json
{
  "name": "mygame",
  "scenes": ["scenes/main.scene.json"],
  "assetsDir": "assets",
  "derivedDir": ".kenga/derived"
}
```

Сцены — JSON с сущностями (Transform, Camera, MeshRenderer, Rigidbody, Collider и т.д.). Примеры в `samples/hello/scenes/`.

## 2. Полноценная игра сейчас?

Нет. Движок в стадии раннего прототипа. Работает:

- ECS, сцены (JSON), рендер Ebiten (wireframe glTF или тестовый треугольник)
- Физика: гравитация, отскок от пола (y=0). Коллизии только с полом.
- Импорт glTF, редактор (Hierarchy, Inspector, Content, Play/Stop, Save), WASM (TinyGo → `game.wasm`)

Нет: нормального 3D, коллизий объект–объект, ввода, аудио, UI, префабов в игре. WebGPU — заглушка.

Итого: можно собрать демо (сцена, меши, падающий объект, WASM), но не игру с геймплеем и меню.

## 3. Что понадобится

Сейчас: Go 1.22+, при WASM — TinyGo в PATH. Шаги из раздела 1.

Чтобы приблизиться к играбельной игрушке: 3D-рендер (камера, меши не только wireframe), ввод (клава/мышь), коллизии объект–объект, геймплей (счёт, победа/поражение, рестарт).

Для «нормальной» игры: плюс материалы, освещение, аудио, UI, префабы, WebGPU.

## 4. Что могу сделать я (AI)

Могу: чинить баги, менять сцены/конфиги/примеры, писать документацию, реализовывать системы и компоненты ECS, CLI, форматы. Предложить архитектуру (ввод, коллизии).

Нужна твоя проверка: запуск `go build` / `go run`, тестов, выбор приоритетов, как выглядит сцена в редакторе.

Не могу: ставить Go/TinyGo, настраивать ОС, гарантировать сборку у тебя без проверки.

## Памятка «дали ссылку на репо»

1. `git clone https://github.com/GermannM3/GoEngineKenga.git`, `cd GoEngineKenga`, `go mod tidy`
2. `go run ./cmd/kenga import --project samples/hello`
3. `go run ./cmd/kenga run --project samples/hello --scene scenes/main.scene.json --backend ebiten`
4. Редактор: `go run ./cmd/kenga-editor`
5. Подробнее: `CREATING_A_GAME.md`, `README.md`.
