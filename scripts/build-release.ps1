# PowerShell script for building GoEngineKenga releases

param(
    [string]$Version = "1.0.0",
    [switch]$Clean,
    [switch]$Test,
    [switch]$NoArchive
)

Write-Host "Building GoEngineKenga Release v$Version" -ForegroundColor Green

# Clean previous builds
if ($Clean) {
    Write-Host "Cleaning previous builds..." -ForegroundColor Yellow
    if (Test-Path "dist") {
        Remove-Item "dist" -Recurse -Force
    }
}

# Create dist directory
New-Item -ItemType Directory -Force -Path "dist" | Out-Null

# Test if requested
if ($Test) {
    Write-Host "Running tests..." -ForegroundColor Yellow
    & go test ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Tests failed!"
        exit 1
    }
}

# Build for Windows amd64
Write-Host "Building for Windows amd64..." -ForegroundColor Cyan
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

$ldflags = "-X main.version=$Version -X main.buildTime=$(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')"
& go build -ldflags $ldflags -o "dist/kenga-editor-windows-amd64.exe" ./cmd/kenga-editor
& go build -ldflags $ldflags -o "dist/kenga-windows-amd64.exe" ./cmd/kenga

# Build for Windows 386
Write-Host "Building for Windows 386..." -ForegroundColor Cyan
$env:GOARCH = "386"
& go build -ldflags $ldflags -o "dist/kenga-editor-windows-386.exe" ./cmd/kenga-editor
& go build -ldflags $ldflags -o "dist/kenga-windows-386.exe" ./cmd/kenga

# Build for Linux amd64
Write-Host "Building for Linux amd64..." -ForegroundColor Cyan
$env:GOOS = "linux"
$env:GOARCH = "amd64"
& go build -ldflags $ldflags -o "dist/kenga-editor-linux-amd64" ./cmd/kenga-editor
& go build -ldflags $ldflags -o "dist/kenga-linux-amd64" ./cmd/kenga

# Build for Linux 386
Write-Host "Building for Linux 386..." -ForegroundColor Cyan
$env:GOARCH = "386"
& go build -ldflags $ldflags -o "dist/kenga-editor-linux-386" ./cmd/kenga-editor
& go build -ldflags $ldflags -o "dist/kenga-linux-386" ./cmd/kenga

# Build for macOS amd64
Write-Host "Building for macOS amd64..." -ForegroundColor Cyan
$env:GOOS = "darwin"
$env:GOARCH = "amd64"
& go build -ldflags $ldflags -o "dist/kenga-editor-darwin-amd64" ./cmd/kenga-editor
& go build -ldflags $ldflags -o "dist/kenga-darwin-amd64" ./cmd/kenga

# Build for macOS arm64
Write-Host "Building for macOS arm64..." -ForegroundColor Cyan
$env:GOARCH = "arm64"
& go build -ldflags $ldflags -o "dist/kenga-editor-darwin-arm64" ./cmd/kenga-editor
& go build -ldflags $ldflags -o "dist/kenga-darwin-arm64" ./cmd/kenga

# Create archives
if (-not $NoArchive) {
    Write-Host "Creating release archives..." -ForegroundColor Yellow

    $platforms = @(
        "windows-amd64",
        "windows-386",
        "linux-amd64",
        "linux-386",
        "darwin-amd64",
        "darwin-arm64"
    )

    foreach ($platform in $platforms) {
        $zipName = "GoEngineKenga-$Version-$platform.zip"
        $files = @(
            "dist/kenga-editor-$platform.exe",
            "dist/kenga-$platform.exe",
            "README.md",
            "LICENSE"
        )

        # Filter existing files
        $existingFiles = $files | Where-Object { Test-Path $_ }

        if ($existingFiles.Count -gt 0) {
            Compress-Archive -Path $existingFiles -DestinationPath "dist/$zipName" -Force
            Write-Host "Created $zipName" -ForegroundColor Green
        }
    }
}

# Create checksums
Write-Host "Creating checksums..." -ForegroundColor Yellow
Get-ChildItem "dist" -File | ForEach-Object {
    $hash = Get-FileHash $_.FullName -Algorithm SHA256
    "$($hash.Hash) $($_.Name)" | Out-File -FilePath "dist/SHA256SUMS" -Append
}

Write-Host "Release build completed!" -ForegroundColor Green
Write-Host "Files created in dist/ directory:" -ForegroundColor Cyan
Get-ChildItem "dist" -Name