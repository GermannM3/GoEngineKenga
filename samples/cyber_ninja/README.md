# CyberNinja — платформер (сэмпл в репозитории)

Запуск **из корня движка** (так сцена точно подхватится, без путей на D:):

```bash
cd d:\GoEngineKenga
go run ./cmd/kenga run --project samples/cyber_ninja --scene scenes/main.scene.json --backend ebiten
```

Управление: **A/D** или стрелки — движение, **Space/W/Up** — прыжок. ПКМ — orbit-камера.

В консоли при старте должна быть строка: `kenga: scene "CyberNinja" from ... (7 entities)` — тогда загружена именно игра.
