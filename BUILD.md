# Сборка и распространение GoEngineKenga

Как собирать релизы и артефакты для Windows, Linux, macOS.

## Быстрая сборка

### Windows (PowerShell)

```powershell
# Собрать все платформы
.\scripts\build-release.ps1 -Version "1.0.0"

# Собрать только Windows
.\scripts\build-release.ps1 -Version "1.0.0" -Clean -Test

# Создать установщик
.\scripts\build-installer.ps1 -Version "1.0.0"
```

### Linux/macOS (Bash)

```bash
# Собрать все платформы
go run build/build.go

# Или использовать скрипт
chmod +x scripts/build-release.sh
./scripts/build-release.sh -version 1.0.0
```

## Создание релиза

### GitHub Actions

1. Создать git tag:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. Workflow соберёт бинарники, архивы, установщик Windows и опубликует релиз.

### Ручная сборка

```bash
# 1. Собрать все бинарные файлы
go run build/build.go

# 2. Создать установщик (Windows)
.\scripts\build-installer.ps1

# 3. Создать контрольные суммы
Get-ChildItem dist -File | ForEach-Object {
    $hash = Get-FileHash $_.FullName -Algorithm SHA256
    "$($hash.Hash) $($_.Name)" | Out-File -FilePath "dist/SHA256SUMS" -Append
}

# 4. Загрузить файлы на GitHub Releases
```

## Требования

### Windows
- Go 1.24+
- PowerShell 5.1+
- NSIS (для создания установщика)
- 7-Zip (для создания архивов)

### Linux
- Go 1.24+
- tar, gzip
- dpkg (для создания .deb пакетов)

### macOS
- Go 1.24+
- tar, gzip

## Структура релиза

```
dist/
├── kenga-editor-windows-amd64.exe    # Редактор Windows
├── kenga-windows-amd64.exe           # CLI Windows
├── kenga-editor-linux-amd64          # Редактор Linux
├── kenga-linux-amd64                 # CLI Linux
├── kenga-editor-darwin-amd64         # Редактор macOS
├── kenga-darwin-amd64                # CLI macOS
├── GoEngineKenga-1.0.0-windows-amd64.zip
├── GoEngineKenga-1.0.0-linux-amd64.tar.gz
├── GoEngineKenga-1.0.0-darwin-amd64.tar.gz
├── GoEngineKenga-1.0.0-installer.exe # Установщик Windows
└── SHA256SUMS                        # Контрольные суммы
```

## Платформы

GoEngineKenga собирается для следующих платформ:

| ОС       | Архитектура | Формат         |
|----------|-------------|----------------|
| Windows  | amd64       | .exe           |
| Windows  | 386         | .exe           |
| Linux    | amd64       | бинарный       |
| Linux    | 386         | бинарный       |
| macOS    | amd64       | бинарный       |
| macOS    | arm64       | бинарный       |

## WebGPU-бэкенд (требуется C-компилятор)

> ✅ **Рабочий путь** (2026-08): WebGPU-бэкенд стабильно работает при сборке
> **w64devkit 1.23.0 (GCC 14.1)**. Используются локальные копии:
> - `third_party/glfw` — класс окна `GLFW3GoEngineKenga` вместо `GLFW30`
>   (ebiten линкует свой GLFW-форк с тем же классом; без замены второй
>   `glfw.Init` в процессе падает «класс уже существует»);
> - `third_party/webgpu` — cogentcore-биндинг с патчами под wgpu 25+ валидацию
>   (явные stencil/multisample/mip/rowsPerImage, `var<uniform>` в пост-шейдере,
>   раздельные layout'ы bright/blur/composite, error scopes убраны — ошибки
>   валидации падают сразу с полным текстом вместо молчаливой порчи объектов).
>
> Не собирайте GCC новее 14.x: поведение более новых мейджоров не проверено.
> После смены тулчейна делайте `go clean -cache` (несогласованный cgo-кэш даёт
> фантомные паники `storage.rs ... does not exist`).

Бэкенд `-tags webgpu` (PBR, MSAA 4×, тени, spotlight) требует cgo и C-компилятор
GNU-ABI (MinGW-w64). Без него используется Ebiten (software rasterizer).

### Windows: тулчейн без админа (w64devkit)

1. Скачать: <https://github.com/skeeto/w64devkit/releases> (например `w64devkit-1.23.0.zip`, там обычный zip).
2. Распаковать, например в `D:\tools\w64devkit`.
3. В текущем shell добавить `bin` в PATH:

   ```powershell
   $env:PATH = "D:\tools\w64devkit\bin;" + $env:PATH
   $env:CGO_ENABLED = "1"
   ```

4. Собрать и запустить:

   ```powershell
   go build -tags webgpu -o kenga-webgpu.exe ./cmd/kenga
   ./kenga-webgpu.exe run --project samples/cyber_ninja --backend webgpu
   ```

Примечания:
- `github.com/cogentcore/webgpu` **статически линкует** `libwgpu_native`
  (папки `wgpu/lib/<os>/<arch>` в модуле) — отдельный DLL не нужен.
- На этой машине при установке тулчейна может вмешиваться Windows Defender
  (помечает свежесобранные/скачанные бинари). Добавь исключение в Defender
  на каталоги проекта и TEMP, если сборка или запуск блокируются.

### Linux/macOS

`libwgpu_native.a` для linux/macos входит в модуль; нужен системный gcc/clang:

```bash
CGO_ENABLED=1 go build -tags webgpu -o kenga-webgpu ./cmd/kenga
```

## Распространение

Windows: NSIS-установщик, ярлыки. Linux: .deb. macOS: .dmg (планируется). Архивы: .zip (Windows), .tar.gz (Linux/macOS). Обновления: проверка через GitHub API.

Сайт загрузки: `website/index.html`. Определяет ОС и предлагает нужный бинарник.

Безопасность: GPG-подпись релизов, SHA256, сканирование (если настроено).

## Checklist перед релизом

- [ ] Все тесты проходят: `go test ./...`
- [ ] Сборка работает на всех платформах
- [ ] Установщик Windows работает корректно
- [ ] Архивы содержат все необходимые файлы
- [ ] README.md обновлен
- [ ] Версия в коде обновлена
- [ ] Changelog написан
- [ ] Контрольные суммы созданы