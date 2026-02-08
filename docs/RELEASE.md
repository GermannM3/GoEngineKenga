# GitHub Release

GoEngineKenga публикует релизы на GitHub через Actions.

## Как создать релиз

### Вариант 1: Через тег (рекомендуется)

```bash
git tag v0.2.0
git push origin v0.2.0
```

1. CI соберёт бинарники для Windows, Linux, macOS (amd64, arm64 где применимо)
2. Создастся релиз с архивом для каждой платформы
3. Release notes сгенерируются из коммитов

### Вариант 2: Вручную (workflow_dispatch)

1. GitHub → Actions → Release → Run workflow
2. Укажи версию (например `0.2.0`)
3. Создастся **draft** релиз — его можно отредактировать и опубликовать вручную

## Артефакты

- `GoEngineKenga-{version}-windows-amd64.zip` — Windows x64
- `GoEngineKenga-{version}-linux-amd64.tar.gz` — Linux x64
- `GoEngineKenga-{version}-linux-arm64.tar.gz` — Linux ARM64
- `GoEngineKenga-{version}-darwin-amd64.tar.gz` — macOS Intel
- `GoEngineKenga-{version}-darwin-arm64.tar.gz` — macOS Apple Silicon

В каждом архиве: `kenga` (или `kenga.exe`), README.md, LICENSE.

## Версия в бинарнике

```bash
kenga --version
```

При запуске `kenga run` с WebSocket API доступен HTTP-эндпойнт:

```
GET http://127.0.0.1:7777/version
# {"version":"v0.2.0"}
```

Версия подставляется при сборке через `-ldflags "-X main.version=v0.2.0"`.

## Prerelease

Теги с суффиксами `-alpha`, `-beta`, `-rc` создают prerelease (например `v0.2.0-beta.1`).
