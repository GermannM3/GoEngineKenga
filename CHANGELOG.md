# Changelog

Формат: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/). Версии: [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-08

### Добавлено

- **WebGPU-бэкенд**: PBR-материалы с текстурами (albedo/normal/metallic-roughness/emissive, fallback), AlphaMode Opaque/Mask, per-material bind groups
- **Скелетная анимация в WebGPU**: ECS-компонент `Animator` + `SkeletalAnimationSystem`, skinning в vertex shader (`shader_skinned.wgsl`), skinned-проход в shadow map
- **Освещение**: point light в WebGPU (attenuation, PBR BRDF), directional shadow map
- **Orbit-камера в WebGPU** (ПКМ rotate, СКМ pan, scroll zoom)
- **2D**: `Sprite`-компонент и спрайтовый рендерер для Ebiten
- **Аудио в игровом цикле**: `AudioSource` реально играет (WAV/OGG), звуки прыжка и подбора предметов
- **WASM-скриптинг**: достроены хост-функции (`getInputKey`, `getEntityTransform` и др.), сборка скриптов через `kenga script build`
- **Локализация (i18n)**: `engine/gameplay/localization.go` — per-locale загрузка, fallback, `Tr(key, args)`; пакетный API `gameplay.Tr`; HUD игр переведён (ru/en); `docs/INTERNATIONAL.md`
- **ECS**: компонент `Health` (current/max), тайминги кругов для гонок
- **Игра CyberNinja** (samples/cyber_ninja): 3D-платформер — патруль дронов, здоровье/урон с откатом и отбрасыванием, сбор сфер, победа/поражение, рестарт, переход между уровнями (2 уровня)
- **Игра Kart Racing** (samples/kart_racing): 2D-гонки — 3 круга по 16 вейпоинтам, 3 машины с разными характеристиками, боты-соперники, бонусы, тайминги кругов (лучший круг), финишный экран, меню
- **Headless-режим** `--headless`: игровые системы без окна (тесты, CI)
- **Юнит-тесты**: 20 тестов — физика (гравитация, отскок, кинематика, drag), анимация (playback, loop, crossfade, skeleton), геймплей (патруль, урон, победа/поражение, круги карта, боты, локализация)
- Тестовая сборка проверяется CI (`go test ./...` + `go build` + WASM)

### Исправлено

- `Animator.Update`: кроссфейд замерзал навсегда, если исходный клип заканчивался (не-цикличный) — теперь кроссфейд доигрывается, новый клип начинает играть
- Расхождение сущностей при перезагрузке сцены: игровые системы сбрасывают состояние по смене World

### Технологии

Go 1.24+, Ebiten 2.9 (по умолчанию), WebGPU (опционально, CGO), wazero (WASM), TinyGo (сборка скриптов). Платформы: Windows, Linux, macOS, WASM, Mobile.

### Быстрый старт

```bash
go run ./cmd/kenga run --project samples/cyber_ninja --backend ebiten   # CyberNinja
go run ./cmd/kenga run --project samples/kart_racing --backend ebiten   # Kart Racing
go build -tags webgpu -o kenga.exe ./cmd/kenga                          # WebGPU (CGO)
go test ./...                                                            # юнит-тесты
```

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