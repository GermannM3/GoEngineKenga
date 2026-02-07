# GoEngineKenga IDE

IDE для движка GoEngineKenga: **Tauri 2 + React + Monaco Editor**.

## UI

- **Меню:** File, Edit, View, Run, Window, Help
- **Панели:** Explorer (дерево файлов), Editor (Monaco с вкладками), Scene (3D viewport), Inspector, Console
- **Командная строка** (AutoCAD-стиль) внизу — ввод команд `kenga run`, `load_model`, `set_camera` и т.д.
- **Resizable panels** — перетаскивание границ панелей

## Требования

- Node.js 18+
- Rust (для Tauri)
- npm / pnpm / yarn

## Разработка

```bash
cd ide
npm install
npm run tauri dev
```

## Сборка и установщики

```bash
npm run tauri build
```

Результат (в `src-tauri/target/release/bundle/`):
- **Windows:** MSI (`msi/`), EXE/NSIS (`nsis/*-setup.exe`)
- **Linux:** DEB (`deb/`), RPM (`rpm/`), AppImage (`appimage/`)

Скрипты: `../scripts/build-ide.ps1` (Windows), `../scripts/build-ide.sh` (Linux).

## Структура

- `src/` — React UI (Monaco, панели, командная строка)
- `src/components/` — MenuBar, CommandLine, ExplorerPanel, EditorTabs, Viewport, InspectorPanel, ConsolePanel
- `src-tauri/` — Rust backend (Tauri 2, shell plugin для kenga)

## Иконки

Для production: `src-tauri/icons/` — 32x32.png, 128x128.png, icon.ico. Можно сгенерировать из `../logo.jpg` через [tauri-icon](https://github.com/tauri-apps/tauri-icon).
