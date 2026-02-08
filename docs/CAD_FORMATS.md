# Поддерживаемые CAD-форматы

Движок Kenga работает с 3D-моделями в формате **glTF/GLB**. Форматы Autodesk Inventor (.ipt, .iam) — проприетарные, их нужно конвертировать в glTF перед импортом.

## Нативные форматы

| Формат | Расширение | Поддержка |
|--------|------------|-----------|
| glTF 2.0 | .gltf, .glb | ✅ Прямой импорт |

## Форматы Inventor (конвертация)

| Формат | Расширение | Описание |
|--------|------------|----------|
| Inventor Part | .ipt | Модель детали |
| Inventor Assembly | .iam | Сборка (ссылки на .ipt) |

### Вариант 1: kenga convert (рекомендуется)

Встроенная команда конвертации через Autodesk Forge:

```bash
# Задайте переменные окружения (см. aps.autodesk.com)
set FORGE_CLIENT_ID=...
set FORGE_CLIENT_SECRET=...

kenga convert model.ipt -o model.glb
```

Требуется Node.js (для forge-convert-utils). При `kenga import` файлы .ipt/.iam в `assets/` конвертируются автоматически, если заданы учётные данные Forge.

**Цепочка:** IPT/IAM → загрузка в Forge OSS → Model Derivative (SVF) → forge-convert-utils → glTF

### Вариант 2: Ручная конвертация в Inventor

Если установлен Autodesk Inventor:

1. Откройте файл .ipt или .iam
2. Файл → Сохранить как → выберите glTF или GLB (если доступно в вашей версии)
3. Или используйте надстройку [ProtoTech GLTF Exporter](https://apps.autodesk.com/INVNTOR/en/Detail/HelpDoc?appId=6155426903769565235)

### Вариант 3: Node.js (forge-convert-utils)

Установите Node.js и пакет:

```bash
npm install -g forge-convert-utils
```

Сначала загрузите файл в Forge и дождитесь конвертации (через веб-интерфейс или API). Затем:

```bash
forge-convert <urn> --output model.glb
```

Или используйте скрипт из репозитория: `scripts/convert-inventor-to-gltf.ps1`

### Вариант 4: CAD Exchanger (платный)

[CAD Exchanger Lab](https://cadexchanger.com/) — десктопное приложение для конвертации 30+ форматов. Открывает .ipt/.iam и экспортирует в glTF.

### Вариант 5: FreeCAD + InventorLoader (только .ipt)

Для одиночных деталей (.ipt):

1. Установите FreeCAD
2. Добавьте Addon InventorLoader
3. Импортируйте .ipt → экспортируйте в glTF

Сборки (.iam) могут работать некорректно.

## Pipeline импорта в Kenga

```
[.ipt / .iam] → конвертация → [.gltf / .glb] → kenga import → derived/
```

После конвертации положите .gltf/.glb в папку `assets/` проекта и выполните:

```bash
kenga import --project <path>
```

## Скрипт конвертации

Скрипт `scripts/convert-inventor-to-gltf.ps1` вызывает `kenga convert`:

```powershell
.\scripts\convert-inventor-to-gltf.ps1 -Input model.ipt -Output model.glb
```
