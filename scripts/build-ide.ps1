# Build GoEngineKenga IDE (Tauri + React + Monaco)
# Output: MSI, EXE (NSIS), DEB, RPM, AppImage (platform-dependent)
# Usage: .\scripts\build-ide.ps1 [-Version "0.1.0"]

param(
    [string]$Version = "0.1.0"
)

$ErrorActionPreference = "Stop"
$ideDir = Join-Path $PSScriptRoot ".." "ide"

Write-Host "Building GoEngineKenga IDE v$Version" -ForegroundColor Green
Write-Host "Working directory: $ideDir" -ForegroundColor Cyan

Push-Location $ideDir

try {
    # Install deps if needed
    if (-not (Test-Path "node_modules")) {
        Write-Host "Installing npm dependencies..." -ForegroundColor Yellow
        npm install
    }

    # Update version in package.json and tauri.conf
    (Get-Content "package.json") -replace '"version": "[^"]*"', "`"version`": `"$Version`"" | Set-Content "package.json"
    (Get-Content "src-tauri/tauri.conf.json") -replace '"version": "[^"]*"', "`"version`": `"$Version`"" | Set-Content "src-tauri/tauri.conf.json"

    # Build (Tauri builds platform-specific installers)
    Write-Host "Running tauri build..." -ForegroundColor Yellow
    npm run tauri build

    # Output location
    $bundleDir = Join-Path $ideDir "src-tauri" "target" "release" "bundle"
    if (Test-Path $bundleDir) {
        Write-Host "`nBuild complete! Installers:" -ForegroundColor Green
        Get-ChildItem $bundleDir -Recurse -Include "*.msi","*.exe","*.deb","*.rpm","*.AppImage" | ForEach-Object {
            Write-Host "  $($_.FullName)" -ForegroundColor Cyan
        }
    }
} finally {
    Pop-Location
}
