# Changelog

Формат: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/). Версии: [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-01-24

### Добавлено

- ECS, сцены (JSON), префабы (структура)
- Физика: rigidbody, коллайдеры, гравитация, отскок от пола
- Рендер Ebiten: wireframe glTF, тестовый треугольник. WebGPU — заглушка
- glTF импорт, asset index в `.kenga/`
- Редактор: Hierarchy, Inspector, Content, Play/Stop, Save, импорт
- WASM-скриптинг (TinyGo), горячая перезагрузка
- CLI: `import`, `run`, `script build`
- Плагины: интерфейс, пример в `examples/plugins`
- Аудио, UI, профилировщик, отладчик: заглушки

### Технологии

Go 1.22+, TinyGo (WASM), Ebiten. Платформы: Windows, Linux, macOS.

### Быстрый старт

```bash
go run ./cmd/kenga import --project samples/hello
go run ./cmd/kenga run --project samples/hello --scene scenes/main.scene.json --backend ebiten
go run ./cmd/kenga-editor
```

### Планы

v1.1: WebGPU/3D, коллизии объект–объект, ввод. Дальше: аудио, UI, мультиплеер, мобильные платформы.