# Конвертация IPT/IAM в glTF через Autodesk Forge
# Требует: FORGE_CLIENT_ID, FORGE_CLIENT_SECRET, Node.js
#
# Использование:
#   .\convert-inventor-to-gltf.ps1 -Input model.ipt [-Output model.glb]
#   $env:FORGE_CLIENT_ID="..."
#   $env:FORGE_CLIENT_SECRET="..."

param(
    [Parameter(Mandatory=$true)]
    [string]$Input,
    [string]$Output
)

$kenga = "kenga"
if (Get-Command "kenga" -ErrorAction SilentlyContinue) { }
elseif (Test-Path ".\kenga.exe") { $kenga = ".\kenga.exe" }
elseif (Test-Path "..\kenga.exe") { $kenga = "..\kenga.exe" }
else {
    Write-Error "kenga не найден. Запустите из корня репозитория или добавьте kenga в PATH."
    exit 1
}

$args = @("convert", $Input)
if ($Output) { $args += "-o", $Output }
& $kenga $args
exit $LASTEXITCODE
