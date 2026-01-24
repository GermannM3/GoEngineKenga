# Update version in GoEngineKenga source code

param(
    [Parameter(Mandatory=$true)]
    [string]$Version,

    [switch]$Commit
)

Write-Host "Updating GoEngineKenga version to $Version" -ForegroundColor Green

# Update version in main files
$files = @(
    "cmd/kenga-editor/main.go",
    "cmd/kenga/main.go",
    "build/build.go"
)

foreach ($file in $files) {
    if (Test-Path $file) {
        $content = Get-Content $file -Raw

        # Update version constants
        $content = $content -replace 'version.*=.*".*"' , "version = `"$Version`""

        Set-Content $file $content -Encoding UTF8
        Write-Host "Updated $file" -ForegroundColor Cyan
    }
}

# Update README.md
if (Test-Path "README.md") {
    $content = Get-Content "README.md" -Raw
    $content = $content -replace 'Текущая версия:.*?\*\*', "Текущая версия: **$Version**"
    $content = $content -replace 'Размер:.*?\*\*', "Размер: **~15MB**"
    Set-Content "README.md" $content -Encoding UTF8
    Write-Host "Updated README.md" -ForegroundColor Cyan
}

# Update website
if (Test-Path "website/index.html") {
    $content = Get-Content "website/index.html" -Raw
    $content = $content -replace 'Скачать последнюю версию \(.*?\)', "Скачать последнюю версию ($Version)"
    $content = $content -replace 'Текущая версия:.*?<strong>', "Текущая версия: <strong>$Version</strong>"
    $content = $content -replace 'Дата релиза:.*?<strong>', "Дата релиза: <strong>$(Get-Date -Format 'yyyy-MM-dd')</strong>"
    Set-Content "website/index.html" $content -Encoding UTF8
    Write-Host "Updated website/index.html" -ForegroundColor Cyan
}

if ($Commit) {
    Write-Host "Creating git commit..." -ForegroundColor Yellow
    git add .
    git commit -m "Release version $Version"
    git tag "v$Version"
    Write-Host "Created commit and tag v$Version" -ForegroundColor Green
}

Write-Host "Version update completed!" -ForegroundColor Green
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. Run build scripts: .\scripts\build-release.ps1 -Version $Version" -ForegroundColor White
Write-Host "2. Test the build" -ForegroundColor White
Write-Host "3. Create GitHub release" -ForegroundColor White
Write-Host "4. Push to repository: git push && git push --tags" -ForegroundColor White