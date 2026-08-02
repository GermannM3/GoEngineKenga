# Файловая структура проекта CyberNinja

## Описание всех файлов и папок проекта

### Основные файлы движка (добавленные для поддержки 2D):
- `engine/ecs/world.go` - добавлены компоненты SpriteRenderer и Camera2D
- `engine/scene/scene.go` - поддержка новых компонентов в сценах
- `engine/render/ebiten/sprite_renderer.go` - система отрисовки спрайтов
- `engine/render/ebiten/backend.go` - интеграция 2D рендеринга
- `engine/cli/run.go` - автоматическое переключение в 2D режим

### Проект CyberNinja:
#### Конфигурация:
- `CyberNinja/project.kenga.json` - основная конфигурация проекта

#### Сцены:
- `CyberNinja/scenes/title.scene.json` - титульный экран
- `CyberNinja/scenes/main.scene.json` - основной уровень (City Street)
- `CyberNinja/scenes/level2.scene.json` - второй уровень (Sewers)

#### Ассеты:
##### Фоны:
- `CyberNinja/assets/backgrounds/city_street.png` - фон города
- `CyberNinja/assets/backgrounds/sewers.png` - фон канализации
- `CyberNinja/assets/backgrounds/rooftop.png` - фон крыши
- `CyberNinja/assets/backgrounds/corporate_hq.png` - фон корпоративного офиса

##### Персонажи:
###### Герой:
- `CyberNinja/assets/characters/hero/hero_tpose_front.png` - статичная поза героя
- `CyberNinja/assets/characters/hero/hero_attack_pose.png` - поза атаки
- `CyberNinja/assets/characters/hero/hero_defend_pose.png` - поза защиты

###### Враги:
- `CyberNinja/assets/characters/enemies/drone_scout.png` - дрон-разведчик
- `CyberNinja/assets/characters/enemies/mech_spider.png` - механический паук
- `CyberNinja/assets/characters/enemies/security_bot.png` - охранный робот

###### Боссы:
- `CyberNinja/assets/characters/boss/ninja_boss.png` - босс-ниндзя
- `CyberNinja/assets/characters/boss/final_boss.png` - финальный босс

###### NPC:
- `CyberNinja/assets/characters/npc/mentor_koala.png` - наставник-коала
- `CyberNinja/assets/characters/npc/hacker_kid.png` - хакер-подросток
- `CyberNinja/assets/characters/npc/merchant_bot.png` - торговый робот

##### Предметы:
- `CyberNinja/assets/items/item_set.png` - набор предметов (меч, бонусы и т.д.)

##### Интерфейс:
- `CyberNinja/assets/ui/title_screen.png` - изображение титульного экрана
- `CyberNinja/assets/ui/ui_frame.png` - рамка интерфейса

#### Скрипты:
- `CyberNinja/scripts/game/main.go` - основной игровой скрипт с управлением

#### Документация:
- `CyberNinja/README.md` - описание проекта
- `IMPLEMENTATION_SUMMARY.md` - техническая документация по реализации
- `CYBERNINJA_IMPLEMENTATION.md` - подробное описание проделанной работы
- `run_cyberninja.bat` - скрипт запуска для Windows
- `run_cyberninja.sh` - скрипт запуска для Linux/Mac

## Игровые механики

### Уровень 1 (City Street):
- Игрок начинает на улице города
- Есть платформы для прыжков
- Встречается враг-дрон
- Есть NPC-наставник
- Есть предмет для подбора

### Уровень 2 (Sewers):
- Более сложный уровень в канализации
- Больше платформ
- Более сильный враг - механический паук
- Дополнительные бонусы

## Технические особенности

### Система спрайтов:
- Автоматическое определение необходимости 2D рендеринга
- Поддержка слоёв для правильного отображения
- Поддержка трансформаций (позиция, масштаб, отражения)
- Кеширование текстур для производительности

### Физика:
- Гравитация для всех объектов
- Система коллизий
- Возможность прыжков и перемещения

### Камера:
- 2D камера, следящая за игроком
- Поддержка масштабирования

### Управление:
- Перемещение: A/D или стрелки
- Прыжок: Space
- Система замедления при отпускании клавиш