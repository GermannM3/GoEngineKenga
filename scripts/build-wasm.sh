#!/bin/bash
# Build GoEngineKenga for WebAssembly
# Output: wasm/kenga.wasm, wasm/index.html
# Run: cd wasm && python3 -m http.server 8080

set -e
cd "$(dirname "$0")/.."
mkdir -p wasm

GOOS=js GOARCH=wasm go build -o wasm/kenga.wasm ./cmd/kenga-wasm

GOROOT=$(go env GOROOT)
if [ -f "$GOROOT/lib/wasm/wasm_exec.js" ]; then
  cp "$GOROOT/lib/wasm/wasm_exec.js" wasm/
elif [ -f "$GOROOT/misc/wasm/wasm_exec.js" ]; then
  cp "$GOROOT/misc/wasm/wasm_exec.js" wasm/
fi

echo "Build complete. wasm/kenga.wasm"
echo "Serve: cd wasm && python3 -m http.server 8080"
echo "Open: http://localhost:8080"
