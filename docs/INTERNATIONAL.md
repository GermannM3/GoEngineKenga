# International readiness

Рекомендации для подготовки движка и игр к международному рынку.

## Localization (i18n)

### API

```go
import "goenginekenga/engine/gameplay"

loc := gameplay.NewLocalization("en", "en")  // locale, fallback
loc.LoadDir("locales")                       // locales/en.json, locales/ru.json

// Простая строка
text := loc.Get("menu.start")  // "Start" или "Начать"

// С подстановкой (printf)
text := loc.Tr("score", 100)   // "Score: 100" или "Очки: 100"

// Смена языка
loc.SetLocale("ru")
```

### Формат файлов

`locales/en.json`:
```json
{
  "menu.start": "Start",
  "menu.quit": "Quit",
  "score": "Score: %d",
  "game.over": "Game Over"
}
```

`locales/ru.json`:
```json
{
  "menu.start": "Начать",
  "menu.quit": "Выход",
  "score": "Очки: %d",
  "game.over": "Конец игры"
}
```

### Проект

В `project.kenga.json` можно задать язык по умолчанию:

```json
{
  "name": "MyGame",
  "scenes": ["scenes/main.scene.json"],
  "locale": "en"
}
```

## Чеклист для релиза

- [ ] Все строки интерфейса в локализаторе (никогда не хардкодить)
- [ ] Fallback locale (en) всегда заполнен
- [ ] Тестирование RTL-языков (арабский, иврит) при необходимости
- [ ] Документация API на английском (README, DEVELOPER.md)
- [ ] Лицензия (MIT, Apache) явно указана

## Рекомендуемые локали

| Код | Язык |
|-----|------|
| en | English |
| ru | Русский |
| de | Deutsch |
| es | Español |
| fr | Français |
| zh | 中文 |
| ja | 日本語 |
