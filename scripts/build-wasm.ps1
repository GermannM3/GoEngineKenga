# Build GoEngineKenga for WebAssembly
# Output: wasm/kenga.wasm, wasm/index.html
# Run: cd wasm && python -m http.server 8080

$ErrorActionPreference = "Stop"
$wasmDir = Join-Path $PSScriptRoot ".." "wasm"

if (-not (Test-Path $wasmDir)) {
    New-Item -ItemType Directory -Path $wasmDir | Out-Null
}

Push-Location (Split-Path $PSScriptRoot -Parent)

$env:GOOS = "js"
$env:GOARCH = "wasm"
go build -o (Join-Path $wasmDir "kenga.wasm") ./cmd/kenga-wasm

$goRoot = go env GOROOT
$wasmExec = Join-Path $goRoot "lib" "wasm" "wasm_exec.js"
if (-not (Test-Path $wasmExec)) {
    $wasmExec = Join-Path $goRoot "misc" "wasm" "wasm_exec.js"
}
Copy-Item $wasmExec (Join-Path $wasmDir "wasm_exec.js") -Force

Pop-Location
Write-Host "Build complete. wasm/kenga.wasm"
Write-Host "Serve: cd wasm && python -m http.server 8080"
Write-Host "Open: http://localhost:8080"
