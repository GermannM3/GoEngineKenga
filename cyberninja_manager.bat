@echo off
echo.
echo ========================
echo CyberNinja 2D Платформер
echo ========================
echo.

:menu
echo.
echo Выберите действие:
echo 1. Запустить титульный экран
echo 2. Запустить основной уровень
echo 3. Запустить второй уровень
echo 4. Собрать WASM скрипты
echo 5. Выход
echo.
set /p choice="Введите номер (1-5): "

if "%choice%"=="1" goto title
if "%choice%"=="2" goto main
if "%choice%"=="3" goto level2
if "%choice%"=="4" goto build
if "%choice%"=="5" goto end

echo Неверный выбор
goto menu

:title
echo Запуск титульного экрана...
go run ./cmd/kenga run --project CyberNinja --scene scenes/title.scene.json --backend ebiten
goto menu

:main
echo Запуск основного уровня...
go run ./cmd/kenga run --project CyberNinja --scene scenes/main.scene.json --backend ebiten
goto menu

:level2
echo Запуск второго уровня...
go run ./cmd/kenga run --project CyberNinja --scene scenes/level2.scene.json --backend ebiten
goto menu

:build
echo Сборка WASM скриптов...
if not exist "D:\GoEngineKenga\CyberNinja\.kenga\scripts" mkdir "D:\GoEngineKenga\CyberNinja\.kenga\scripts"
cd /d "D:\GoEngineKenga\CyberNinja\scripts\game"
echo Сборка game.wasm...
tinygo build -o "../..\.kenga\scripts\game.wasm" -target wasm . 
if %ERRORLEVEL% EQU 0 (
    echo WASM скрипт успешно собран!
) else (
    echo Ошибка при сборке WASM скрипта
)
cd /d "D:\GoEngineKenga"
goto menu

:end
echo Спасибо за игру!
pause