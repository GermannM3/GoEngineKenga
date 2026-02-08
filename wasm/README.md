# GoEngineKenga — WebAssembly

3D viewport в браузере (Ebiten на WASM).

## Сборка

```bash
# Windows
.\scripts\build-wasm.ps1

# Linux / macOS
./scripts/build-wasm.sh
```

Создаёт `kenga.wasm` и копирует `wasm_exec.js` из Go.

## Запуск

WASM требует HTTP (не file://):

```bash
cd wasm
python -m http.server 8080
# или: python3 -m http.server 8080
```

Открыть http://localhost:8080

## Разработка

```bash
go run github.com/hajimehoshi/wasmserve@latest ./cmd/kenga-wasm
```

wasmserve запускает сервер с hot-reload.
