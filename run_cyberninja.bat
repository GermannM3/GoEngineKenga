@echo off
echo.
echo ========================
echo CyberNinja 2D Платформер
echo ========================
echo.
echo Выберите уровень:
echo 1. Титульный экран
echo 2. Основной уровень (City Street)
echo 3. Второй уровень (Sewers)
echo.
set /p choice="Введите номер (1-3): "

if "%choice%"=="1" goto title
if "%choice%"=="2" goto main
if "%choice%"=="3" goto level2

echo Неверный выбор
goto end

:title
echo Запуск титульного экрана...
go run ./cmd/kenga run --project CyberNinja --scene scenes/title.scene.json --backend ebiten
goto end

:main
echo Запуск основного уровня...
go run ./cmd/kenga run --project CyberNinja --scene scenes/main.scene.json --backend ebiten
goto end

:level2
echo Запуск второго уровня...
go run ./cmd/kenga run --project CyberNinja --scene scenes/level2.scene.json --backend ebiten
goto end

:end
pause