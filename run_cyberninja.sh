#!/bin/bash

echo ""
echo "========================"
echo "CyberNinja 2D Платформер"
echo "========================"
echo ""

echo "Выберите уровень:"
echo "1. Титульный экран"
echo "2. Основной уровень (City Street)"
echo "3. Второй уровень (Sewers)"
echo ""

read -p "Введите номер (1-3): " choice

case $choice in
    1)
        echo "Запуск титульного экрана..."
        go run ./cmd/kenga run --project CyberNinja --scene scenes/title.scene.json --backend ebiten
        ;;
    2)
        echo "Запуск основного уровня..."
        go run ./cmd/kenga run --project CyberNinja --scene scenes/main.scene.json --backend ebiten
        ;;
    3)
        echo "Запуск второго уровня..."
        go run ./cmd/kenga run --project CyberNinja --scene scenes/level2.scene.json --backend ebiten
        ;;
    *)
        echo "Неверный выбор"
        ;;
esac