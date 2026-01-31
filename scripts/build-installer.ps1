# Build GoEngineKenga Windows installer using NSIS

param(
    [string]$Version = "1.0.0",
    [string]$NsisPath = "C:\Program Files (x86)\NSIS\makensis.exe"
)

Write-Host "Building GoEngineKenga Windows Installer v$Version" -ForegroundColor Green

# Find makensis: -NsisPath, then Program Files, then PATH
if (-not (Test-Path $NsisPath)) {
    $NsisPath = "C:\Program Files\NSIS\makensis.exe"
}
if (-not (Test-Path $NsisPath)) {
    $NsisPath = "C:\Program Files (x86)\NSIS\makensis.exe"
}
if (-not (Test-Path $NsisPath)) {
    $exe = Get-Command makensis -ErrorAction SilentlyContinue
    if ($exe) { $NsisPath = $exe.Source }
}
if (-not (Test-Path $NsisPath)) {
    Write-Warning "NSIS not found. Install from https://nsis.sourceforge.io/ or: winget install NSIS.NSIS"
    Write-Host "Or specify: -NsisPath 'C:\path\to\makensis.exe'" -ForegroundColor Yellow
    exit 1
}

# Check if dist directory exists with built binaries
if (-not (Test-Path "dist")) {
    Write-Error "dist directory not found. Run build-release.ps1 first."
    exit 1
}

# Copy installer files
Write-Host "Preparing installer files..." -ForegroundColor Cyan
Copy-Item "installer\installer.nsi" "dist\" -Force

# Build installer
Write-Host "Building installer..." -ForegroundColor Cyan
Push-Location "dist"
try {
    & $NsisPath "installer.nsi"

    if ($LASTEXITCODE -eq 0) {
        $installerName = "GoEngineKenga-$Version-setup.exe"
        if (-not (Test-Path $installerName)) { $installerName = "GoEngineKenga-$Version-installer.exe" }
        if (Test-Path $installerName) {
            Write-Host "Installer created successfully: $installerName" -ForegroundColor Green

            # Create checksum
            $hash = Get-FileHash $installerName -Algorithm SHA256
            "$($hash.Hash) $installerName" | Out-File -FilePath "SHA256SUMS" -Append

            Write-Host "SHA256 checksum added to SHA256SUMS" -ForegroundColor Cyan
        } else {
            Write-Error "Installer executable not found after build"
            exit 1
        }
    } else {
        Write-Error "NSIS build failed with exit code $LASTEXITCODE"
        exit 1
    }
} finally {
    Pop-Location
}

Write-Host "Installer build completed!" -ForegroundColor Green