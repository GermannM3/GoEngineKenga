# GoEngineKenga — Mobile (Android / iOS)

Сборка через `ebitenmobile bind` — создаёт shared library (.aar для Android, .xcframework для iOS) для интеграции в нативный проект.

## Требования

- `ebitenmobile`: `go install github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@latest`
- Android: ANDROID_HOME, JDK
- iOS: Xcode, macOS

## Android

```bash
# Установка (macOS)
export ANDROID_HOME=~/Library/Android/sdk
export PATH=$ANDROID_HOME/../jbr/Contents/Home/bin:$PATH

# Сборка
./scripts/build-mobile-android.sh
# или на Windows: .\scripts\build-mobile-android.ps1
```

Результат: `mobile/kenga.aar`

Интеграция в Android Studio:
1. Добавить `kenga.aar` как модуль
2. В Activity.onCreate вызвать `Seq.setContext(getApplicationContext())`
3. Добавить `EbitenView` (пакет `com.goenginekenga.mobile`) на экран

## iOS

```bash
ebitenmobile bind -target ios -o mobile/GoEngineKenga.xcframework .
```

Результат: `mobile/GoEngineKenga.xcframework`

В Xcode: изменить `ViewController` на `GoenginekengaEbitenViewController` из фреймворка.

## Детали

- Пакет `mobile/` реализует 3D-viewport с дефолтной сценой (куб)
- Без WebSocket, file watcher, asset loading — минимальный runtime
- [Документация Ebitengine Mobile](https://ebitengine.org/en/documents/mobile.html)
