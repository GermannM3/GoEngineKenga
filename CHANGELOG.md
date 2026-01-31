# Changelog

Формат: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/). Версии: [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-01

### Добавлено

- ECS, сцены (JSON), префабы
- Физика: rigidbody, коллайдеры (AABB, sphere, box-sphere), гравитация
- 3D-рендер (software): растеризация, текстуры, освещение, glTF
- glTF импорт, asset pipeline (`.kenga/`)
- Редактор (Fyne): Hierarchy, Inspector, Content, Console, Scene view, Play/Stop, Save, ярлыки, меню
- WebSocket API для внешнего управления: `load_model`, `clear_scene`, `set_camera`, `set_transform`, `set_trajectory`, `add_trajectory_point`, `set_joint`, `start_dispensing`/`stop_dispensing`
- Траектории: компонент и рендер линий по точкам
- Установщик на Go: один exe (`scripts/make-setup.bat`), встроенные файлы; NSIS-вариант и Linux (.sh, .deb) — в docs/INSTALLER.md
- Документация: CAD-режим (docs/CAD.md), установщики (docs/INSTALLER.md), честная оценка (docs/ОЦЕНКА_ДВИЖКОВ.md)
- CLI: `kenga new`, `kenga run`, `kenga import`, `kenga script build`; флаги `--ws-port`, `--no-ws`
- WASM-скриптинг (TinyGo), плагины, аудио, частицы, процедурная генерация, pathfinding (engine/ai)

### Технологии

Go 1.22+, Ebiten, Fyne (редактор). Платформы: Windows, Linux, macOS.

### Быстрый старт

```bash
go run ./cmd/kenga import --project samples/hello
go run ./cmd/kenga run --project samples/hello --scene scenes/main.scene.json --backend ebiten
# Редактор (требует CGO на Windows): go run ./cmd/kenga-editor
# Установщик: scripts\make-setup.bat  →  dist\GoEngineKenga-Setup.exe
```