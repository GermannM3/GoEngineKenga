# Setup dev environment for GoEngineKenga (run as Administrator)
# Run: powershell -ExecutionPolicy Bypass -File scripts/setup-dev-env.ps1

$ErrorActionPreference = "Stop"

$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "Run PowerShell as Administrator." -ForegroundColor Red
    exit 1
}

Write-Host "=== GoEngineKenga: installing tools ===" -ForegroundColor Green

if (-not (Get-Command choco -ErrorAction SilentlyContinue)) {
    Write-Host "Installing Chocolatey..." -ForegroundColor Cyan
    Set-ExecutionPolicy Bypass -Scope Process -Force
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
    Invoke-Expression ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
} else {
    Write-Host "Chocolatey already installed." -ForegroundColor Green
}

$packages = @("git", "go", "nsis")
foreach ($p in $packages) {
    Write-Host "Installing $p..." -ForegroundColor Cyan
    choco install $p -y --no-progress
}
Write-Host "Installing mingw (for CGO)..." -ForegroundColor Cyan
choco install mingw -y --no-progress 2>$null
if ($LASTEXITCODE -ne 0) { Write-Host "mingw: install manually if you need CGO." -ForegroundColor Yellow }

$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")

Write-Host "Done. Next: .\scripts\build-release.ps1 -Version 1.0.0 -Clean -NoArchive" -ForegroundColor Green
Write-Host "Then: .\scripts\build-installer.ps1 -Version 1.0.0" -ForegroundColor Green
