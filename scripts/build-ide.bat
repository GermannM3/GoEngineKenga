@echo off
setlocal
cd /d "%~dp0..\ide"

echo Building GoEngineKenga IDE...
call npm.cmd install
if errorlevel 1 exit /b 1

if not exist "src-tauri\icons\icon.ico" (
    echo Generating icons from logo...
    call npx.cmd tauri icon "..\logo.jpg"
    if errorlevel 1 (
        echo Icon generation failed. Create src-tauri\icons\icon.ico manually.
        exit /b 1
    )
)
call npm.cmd run tauri build
if errorlevel 1 (
    echo tauri build failed
    exit /b 1
)

echo.
echo Build complete! Check ide\src-tauri\target\release\bundle\
endlocal
