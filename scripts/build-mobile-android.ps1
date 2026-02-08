# Build GoEngineKenga for Android (ebitenmobile bind)
# Requires: ANDROID_HOME, ebitenmobile, JDK
#
#	go install github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@latest
#	$env:ANDROID_HOME = "C:\Users\...\AppData\Local\Android\Sdk"
#
# Output: mobile/kenga.aar

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path $PSScriptRoot -Parent
Push-Location $projectRoot

$outPath = Join-Path $projectRoot "mobile" "kenga.aar"
ebitenmobile bind -target android -javapkg com.goenginekenga -o $outPath ./mobile

Pop-Location
Write-Host "Android AAR: $outPath"
