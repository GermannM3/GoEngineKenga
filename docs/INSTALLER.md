# Установщики GoEngineKenga

Установщики: один exe (без NSIS) или NSIS-мастер. Ярлыки, удаление через «Установка и удаление программ».

## Windows: один exe без NSIS (рекомендуется)

Сборка одного установочного exe без PowerShell и NSIS:

```batch
scripts\make-setup.bat
```

Результат: `dist\GoEngineKenga-Setup.exe`. Пользователь запускает его один раз; все файлы встроены в exe.

- **CLI** (kenga) — всегда включается, собирается без CGO.
- **Редактор** (kenga-editor) — включается, если при сборке доступны CGO и MinGW. Иначе — только CLI. Для полного пакета: установите MinGW (`choco install mingw`), затем запустите `make-setup.bat`.

## Windows: NSIS-установщик

### Требования для сборки установщика

- **PowerShell** (от имени администратора для установки зависимостей).
- **Go** — [golang.org](https://go.dev/dl/) или `choco install go`.
- **NSIS** — [nsis.sourceforge.io](https://nsis.sourceforge.io/) или `choco install nsis`.
- Для сборки редактора (Fyne) нужен **CGO** и компилятор C (MinGW): `choco install mingw`.

### Установка всего окружения одной командой

Запустите **PowerShell от имени администратора** (правый клик по PowerShell → «Запуск от имени администратора»), затем:

```powershell
cd D:\GoEngineKenga
Set-ExecutionPolicy Bypass -Scope Process -Force
.\scripts\setup-dev-env.ps1
```

Скрипт установит Chocolatey (если нет), затем git, go, NSIS, MinGW.

### Сборка установщика

1. Собрать бинарники и положить их в `dist/`:

   ```powershell
   .\scripts\build-release.ps1 -Version "1.0.0" -Clean -NoArchive
   ```

   Если редактор не соберётся из‑за CGO/OpenGL, в `dist/` должны быть хотя бы:
   - `kenga-editor-windows-amd64.exe`
   - `kenga-windows-amd64.exe`

2. Собрать exe-установщик:

   ```powershell
   .\scripts\build-installer.ps1 -Version "1.0.0"
   ```

   Результат: `dist\GoEngineKenga-1.0.0-setup.exe`.

### Что делает установщик

- Страницы мастера: приветствие, выбор каталога, компоненты, установка, завершение.
- Компоненты: основная установка, ярлык на рабочем столе, пункты в меню «Пуск», добавление в PATH (через setx, лимит 1024 символа).
- Регистрация в «Установка и удаление программ», App Paths для `kenga`/`kenga-editor`.
- В конце — опция «Запустить редактор».

---

## Linux

### Вариант 1: скрипт install-linux.sh

После сборки релиза (`./scripts/build-release.sh` или аналог) запустите:

```bash
chmod +x scripts/install-linux.sh
./scripts/install-linux.sh
```

По умолчанию ставит в `/usr/local`. Чтобы ставить в свой каталог:

```bash
INSTALL_PREFIX="$HOME/.local" ./scripts/install-linux.sh
```

### Вариант 2: пакет .deb

```bash
./scripts/create-deb.sh -version 1.0.0 -arch amd64
```

Получится файл `goenginekenga_1.0.0_amd64.deb`. Установка:

```bash
sudo dpkg -i goenginekenga_1.0.0_amd64.deb
```

---

## Примечание про softhub_x64.exe

Файл `softhub_x64.exe` из папки «Загрузки» не анализировался и не запускался (безопасность и лицензии). Установщик GoEngineKenga сделан по общему образцу современных инсталлеров: мастер, выбор каталога, опции (ярлыки, PATH), регистрация в системе, корректное удаление.
