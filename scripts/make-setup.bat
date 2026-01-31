@echo off
REM Build one installer exe (no NSIS). Fills embed folder then builds.
cd /d "%~dp0.."

if not exist dist mkdir dist
if not exist cmd\kenga-installer\embed mkdir cmd\kenga-installer\embed

echo Building kenga CLI...
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -o cmd\kenga-installer\embed\kenga-windows-amd64.exe .\cmd\kenga
if errorlevel 1 exit /b 1

echo Copying README, LICENSE, samples into embed...
copy /Y README.md cmd\kenga-installer\embed\ >nul
copy /Y LICENSE cmd\kenga-installer\embed\ >nul
if not exist cmd\kenga-installer\embed\samples mkdir cmd\kenga-installer\embed\samples
xcopy /E /I /Y samples\hello cmd\kenga-installer\embed\samples\hello >nul

echo Building installer (single exe)...
go build -o dist\GoEngineKenga-Setup.exe .\cmd\kenga-installer
if errorlevel 1 exit /b 1

echo.
echo Done. Single installer: dist\GoEngineKenga-Setup.exe
echo User runs it once; no other files needed.
pause
