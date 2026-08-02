# Atom & Moskvich Racing

Аркадный картинг на GoEngineKenga (вид сверху).

## Запуск

Из корня репозитория:

```bash
go run ./cmd/kenga --project samples/kart_racing run
```

Ассеты (если нет `assets/cars/*.png`):

```bash
cd samples/kart_racing
go run scripts/create_placeholders.go
```

## Как играть

1. Меню: **1 / 2 / 3** — Atom / M70 / M90, **Enter** — старт.
2. После «3…2…1…GO» — гонка на 3 круга.
3. Управление:
   - **W/S** — газ / тормоз
   - **A/D** — руль
   - **Shift** — дрифт
   - **Space** — бонус (после подбора жёлтого бокса)
4. Финиш → **Enter** — снова меню.

## Что есть в движке

- `SpriteRenderer` с поворотом по `Rotation.Z`
- `Kart` / `PowerUpPickup` в ECS
- `gameplay.KartSystem` — меню, countdown, AI по waypoints овала, лапы, бонусы
- Процедурный овал-трек (трава + асфальт + разметка)
