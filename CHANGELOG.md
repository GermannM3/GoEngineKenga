# Changelog

Формат: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/). Версии: [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Исправлено

- **WebGPU-бэкенд запускается и стабильно рендерит** (был полностью нерабочим):
  конфликт класса окна GLFW30 между ebiten-форком и go-gl/glfw (локальный
  `third_party/glfw` с уникальным классом), 6 нарушений валидации wgpu 25+
  (stencil/multisample/mip/rowsPerImage/uniform-адресное пространство/DOF-цикл),
  раздельные layout'ы bright/blur/composite пост-процесса (auto-layout bright
  не содержал bloom+юниформ — пост-процесс молча отключался), освобождение
  инстанс-буферов до Submit, error scopes убраны из биндинга — ошибки валидации
  теперь падают сразу с полным текстом. Требования к тулчейну: BUILD.md.

### Добавлено

- **Геймпад**: стандартная раскладка в `input.State` (PadA..PadRight, оси стиков/триггеров, deadzone); Ebiten-бэкенд — кроссплатформенный API ebiten, WebGPU — XInput через syscall (без cgo); в CyberNinja: движение левым стиком/крестовиной (аналоговая скорость), прыжок — A, атака — X
- **Скриптовые компоненты на сущностях (`engine/scripts`)**: аналог MonoBehaviour — поведение регистрируется по имени в `init()` пакета игры (`scripts.Register`), сцена ссылается через `ecs.Script{Name, Params}`; жизненный цикл OnStart/OnUpdate ведёт `scripts.System`. Игровая логика теперь живёт в проекте игры, а не в движке (демо: `samples/cyber_ninja/scripts` — spinner/patrol_x/bob; кристаллы в сцене вращаются скриптом)
- **Вертикальный срез CyberNinja (WebGPU desktop)**: бой, ИИ врагов, следящая камера, звук и обратная связь
  - **Графический HUD в WebGPU**: quad-pipeline с альфа-блендингом + атлас `basicfont.Face7x13`; полоса здоровья, строки HUD, экраны победы/поражения (`Frame.HUD` → `render.BuildHUD`, лейаут покрыт тестами)
  - **Ввод в WebGPU-бэкенде**: клавиатура/мышь через glfw → `input.State` (раньше игра в WebGPU не получала ввод вообще)
  - **CharacterController** (ECS + runtime): кинематическое движение, `Grounded` по raycast'у вниз, прыжок только с земли; параметры из сцены
  - **Следящая камера**: `Camera.FollowID/FollowOffset/FollowLerp` — плавное слежение за игроком (play-режим); orbit остаётся для редактирования
  - **Ближний бой**: компонент `Attacker` (радиус/сектор/кулдаун/knockback), урон+i-frames через `Health`, смерть удаляет сущность (`World.RemoveEntity`)
  - **FSM врагов**: Patrol→Chase→Attack на `ai.StateMachine`, параметры из `EnemyBrain` (scene JSON)
  - **Hazard-зоны**: урон внутри AABB по имени `Hazard*`
  - **Джус**: тряска камеры (`runtime.CameraShake`) на удар/урон; звуки hit/hurt/death.wav (PlayOneShot)
  - **HUD без оверлея**: `Frame.HUDText` показывается в заголовке окна WebGPU
  - Сцены: +3 врага, 8 сфер, 2 «лавовые» зоны; локали `hud.kills`
- **Point light shadows в WebGPU**: cubemap-тени через depth-array (6 граней, 512²), линейная глубина через `frag_depth`, сравнение с bias; **skinned-меши тоже бросают point-тени** (отдельный skinned-шейдер)
- **Spot-тени в WebGPU**: перспективная 2D-карта прожектора (1024²), sampling по `spot_light_view_proj` в spotlight-блоке
- **DOF (tilt-shift) в пост-процессе**: переменный радиальный блюр по фокусной линии UV (без depth-буфера), параметры dof_strength/focus_y/focus_range
- **IBL/отражения в WebGPU**: процедурные env- (64²) и irradiance- (8²) кубомапы без ассетов; диффузный IBL + грубые зеркальные отражения окружения
- **Пост-процесс в WebGPU**: HDR-конвейер (сцена → RGBA16Float offscreen), bloom (bright-extract + separable gaussian 9-tap на полуразрешении) + ACES-тонмаппинг + виньетка в composite-проходе
- **MSAA 4× в WebGPU**: multisample color/depth текстуры + resolve в offscreen-таргет — сглаженные края, корректная сортировка (в main pass появился depth-буфер)
- **Spotlight (прожектор)**: новый тип света `kind=spot` (inner/outer углы конуса, направление из rotation) — в WebGPU (WGSL cone-attenuation) и в soft-растеризаторе (Ebiten); в сцену CyberNinja добавлены демо-прожектор и point-свет
- **Тулчейн WebGPU задокументирован** (BUILD.md: w64devkit, CGO_ENABLED, сборка `-tags webgpu`)

### Исправлено

- **ImportAll терял авторские ассеты**: `.mesh.json`/`.material.json` выкидывались из индекса при re-import (неизвестный тип) — ссылки сцен на меши ломались. Теперь они регистрируются как ассеты со стабильным ID из `.meta`; ID мешей CyberNinja зафиксированы
- **WebGPU-бэкенд не компилировался** (cgo): исправлены предсуществующие ошибки (views[i], err; getMeshMaterial(&mr, ...); лишний импорт) — теперь `go build -tags webgpu` собирается, бэкенд запускается
- **Soft-рендерер ничего не рисовал**: в `rasterizeTriangle` была перевёрнута ориентация теста попадания в треугольник — фиксируется на `w0<=0 && w1<=0 && w2<=0`, после чего меши реально появляются на экране
- **Сцены CyberNinja рендерили вырожденные треугольники**: все `meshRenderer` в `main`/`level2` указывали на один 3-вершинный треугольник вместо кубов. Теперь объекты отрисовываются как объёмные цветные боксы, соответствующие их коллайдерам
- **Освещение soft-рендера**: к диффузному (Lambert) добавлены полусферический (hemisphere) ambient — грани, обращённые вверх, светлее нижних — и мягкий спекуляр (Blinn-Phong); поднята базовая ambient-засветка. Плоские боксы читаются как объёмные

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