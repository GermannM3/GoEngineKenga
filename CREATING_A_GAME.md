# Создание игры на GoEngineKenga

## Обзор возможностей

GoEngineKenga — полнофункциональный движок для создания 2D и 3D игр на Go.

### Что реализовано

| Система | Возможности | Статус |
|---------|-------------|--------|
| **3D Рендер** | Software растеризация, Z-buffer, текстуры | ✅ |
| **Освещение** | Ambient, Directional, Point lights | ✅ |
| **Частицы** | Эмиттеры, физика, цвет/размер по времени | ✅ |
| **Анимация** | Skeletal, keyframe, sprite | ✅ |
| **Физика** | Rigidbody, коллизии, импульсы | ✅ |
| **Физика воды** | Waves, buoyancy, ship physics | ✅ |
| **AI** | A* pathfinding, behavior trees, FSM | ✅ |
| **Процедурка** | Perlin noise, dungeon BSP, heightmaps | ✅ |
| **Шейдеры** | Toon, psychedelic, glitch, fog | ✅ |
| **Деформация** | Wave, twist, bend, melt | ✅ |
| **Аудио** | WAV/MP3/OGG, 3D spatial, FFT | ✅ |
| **UI** | Button, Label, Panel | ✅ |
| **Ввод** | Keyboard, mouse, scroll | ✅ |
| **Сцены** | JSON, ECS, prefabs | ✅ |
| **Скрипты** | WASM (TinyGo) | ✅ |

## Быстрый старт

### 1. Создание проекта

```bash
# Из корня движка
go run ./cmd/kenga new mygame --template default

# Доступные шаблоны: default, platformer, topdown
```

### 2. Структура проекта

```
mygame/
├── project.kenga.json       # Конфигурация
├── scenes/
│   └── main.scene.json      # Главная сцена
├── assets/
│   ├── models/              # 3D модели (.glb, .gltf)
│   ├── textures/            # Текстуры (.png, .jpg)
│   ├── audio/               # Звуки (.wav, .mp3, .ogg)
│   └── sprites/             # 2D спрайты
└── scripts/
    └── game/
        └── main.go          # WASM скрипты
```

### 3. Запуск

```bash
cd mygame
go run ../cmd/kenga run --project . --scene scenes/main.scene.json --backend ebiten
```

## Примеры механик

### Пиратская игра: корабль на волнах

```go
package main

import (
    "goenginekenga/engine/physics"
    emath "goenginekenga/engine/math"
)

func main() {
    // Создать океан с волнами
    ocean := physics.NewOceanWaves()
    ocean.AddWave(emath.Vec2{X: 1, Y: 0}, 50, 1.5, 0.5, 5)   // Основная волна
    ocean.AddWave(emath.Vec2{X: 0.7, Y: 0.7}, 30, 0.8, 0.3, 4) // Вторичная
    
    // Создать корабль
    ship := physics.NewShip(emath.Vec3{X: 0, Y: 0, Z: 0})
    ship.SailArea = 80  // Площадь парусов
    
    // Игровой цикл
    for {
        dt := getDeltaTime()
        
        // Управление рулём
        ship.SetRudder(getInputAxis()) // -1 .. 1
        
        // Ветер
        wind := emath.Vec2{X: 5, Y: 2}
        
        // Обновить физику
        ocean.Update(dt)
        ship.Update(dt, ocean, wind)
        
        // ship.Position, ship.Rotation теперь обновлены
    }
}
```

### Система частиц: взрыв

```go
package main

import (
    "goenginekenga/engine/particles"
    emath "goenginekenga/engine/math"
)

func createExplosion(pos emath.Vec3) *particles.System {
    ps := particles.NewSystem(500)
    
    // Основной взрыв
    explosion := particles.NewExplosionEmitter(pos)
    ps.AddEmitter(explosion)
    
    // Дым после взрыва
    smoke := particles.NewSmokeEmitter(pos)
    smoke.Rate = 10
    ps.AddEmitter(smoke)
    
    // Искры
    sparks := particles.NewSparkEmitter(pos)
    ps.AddEmitter(sparks)
    
    return ps
}
```

### AI: патрулирующий NPC

```go
package main

import (
    "goenginekenga/engine/ai"
    emath "goenginekenga/engine/math"
)

func createPatrollingEnemy(navGrid *ai.NavGrid) *ai.Agent {
    agent := ai.NewAgent(emath.Vec3{X: 10, Y: 0, Z: 10}, 3.0)
    agent.NavGrid = navGrid
    
    // Точки патруля
    waypoints := []emath.Vec3{
        {X: 10, Y: 0, Z: 10},
        {X: 20, Y: 0, Z: 10},
        {X: 20, Y: 0, Z: 20},
        {X: 10, Y: 0, Z: 20},
    }
    
    // State machine
    sm := ai.NewStateMachine(agent)
    
    // Состояние патруля
    sm.AddState(&ai.State{
        Name: "patrol",
        OnUpdate: func(a *ai.Agent, dt float32) {
            a.Update(dt)
        },
        Transitions: []ai.Transition{
            {
                Condition: func(a *ai.Agent) bool {
                    return playerInRange(a, 15)
                },
                NextState: "chase",
            },
        },
    })
    
    // Состояние преследования
    sm.AddState(&ai.State{
        Name: "chase",
        OnEnter: func(a *ai.Agent) {
            a.Speed = 5.0 // Быстрее при преследовании
        },
        OnUpdate: func(a *ai.Agent, dt float32) {
            a.MoveTo(getPlayerPosition())
            a.Update(dt)
        },
        Transitions: []ai.Transition{
            {
                Condition: func(a *ai.Agent) bool {
                    return !playerInRange(a, 20)
                },
                NextState: "patrol",
            },
        },
    })
    
    sm.SetState("patrol")
    return agent
}
```

### Процедурный мир: острова

```go
package main

import "goenginekenga/engine/procgen"

func generatePirateWorld(seed int64) *procgen.WorldMap {
    world := procgen.NewWorldMap(512, 512)
    world.GenerateArchipelago(seed, 15, 0.4) // 15 островов
    
    // Каждый остров имеет:
    // - CenterX, CenterY
    // - Radius
    // - Biome: "tropical", "desert", "volcanic", etc.
    
    for _, island := range world.Islands {
        switch island.Biome {
        case "tropical":
            // Разместить пальмы, пляжи
        case "volcanic":
            // Разместить лаву, дым
        }
    }
    
    return world
}
```

### Аудио-реактивный эффект

```go
package main

import (
    "goenginekenga/engine/audio"
    "goenginekenga/engine/render"
    emath "goenginekenga/engine/math"
)

func createAudioReactiveScene() {
    analyzer := audio.NewAudioAnalyzer(44100, 2048)
    effect := audio.NewAudioReactiveEffect(analyzer)
    
    // Круговой визуализатор
    circular := audio.NewCircularVisualizer(analyzer, 64)
    circular.BaseRadius = 100
    
    // Деформация меша от музыки
    deformer := render.NewSpaceDeformer()
    
    for {
        // Анализировать аудио
        analyzer.PushSamples(getAudioSamples())
        analyzer.Analyze()
        
        // Получить данные
        bassScale := effect.GetBassScale()
        isBeat := analyzer.BeatDetected
        
        if isBeat {
            // Пульс на бите
            deformer.AddDeformation(render.PulseDeform(0.3, 4))
        }
        
        // Деформация от баса
        deformer.AddDeformation(render.WaveDeform(
            float32(analyzer.Bass) * 0.5,  // Амплитуда
            2.0,                            // Частота
            float32(analyzer.Mid) * 2.0,    // Скорость от mid
        ))
        
        // Визуализация спектра
        points := circular.GetPoints(screenWidth/2, screenHeight/2)
        // Рисовать points...
    }
}
```

### Деформация пространства: сюрреализм

```go
package main

import (
    "goenginekenga/engine/render"
    emath "goenginekenga/engine/math"
)

func createSurrealWorld() *render.SpaceDeformer {
    deformer := render.NewSpaceDeformer()
    
    // Волны пространства
    deformer.AddDeformation(render.WaveDeform(0.5, 1.0, 0.5))
    
    // Скручивание реальности
    deformer.AddDeformation(render.TwistDeform(0.1))
    
    // Шумовая деформация
    deformer.AddDeformation(render.NoiseDeform(0.3, 0.5))
    
    // Притяжение к странным точкам
    deformer.AddDeformation(render.SphereDeform(
        emath.Vec3{X: 0, Y: 5, Z: 0},
        3.0,  // Радиус
        0.5,  // Сила
    ))
    
    return deformer
}
```

### Шейдеры: психоделический эффект

```go
package main

import (
    "goenginekenga/engine/shader"
    "image/color"
)

func setupPsychedelicShader() *shader.Shader {
    s := shader.NewShader("psychedelic")
    
    // Настройка
    s.SetUniform("intensity", float32(0.8))
    
    // Использовать встроенный шейдер
    s.FragmentFunc = shader.PsychedelicShader
    
    return s
}

func setupToonShader() *shader.Shader {
    s := shader.NewShader("toon")
    
    s.SetUniform("levels", float32(4))
    s.SetUniform("lightDir", emath.Vec3{X: 0.5, Y: 1, Z: 0.3})
    s.FragmentFunc = shader.ToonFragmentShader
    
    return s
}
```

## Сцена в JSON

```json
{
    "name": "Pirate Cove",
    "entities": [
        {
            "name": "PlayerShip",
            "components": {
                "transform": {
                    "position": {"x": 0, "y": 0, "z": 0},
                    "rotation": {"x": 0, "y": 0, "z": 0}
                },
                "meshRenderer": {
                    "mesh": "models/ship.glb",
                    "material": "materials/wood.mat"
                },
                "rigidbody": {
                    "mass": 5000,
                    "useGravity": false
                }
            }
        },
        {
            "name": "Ocean",
            "components": {
                "transform": {
                    "position": {"x": 0, "y": -0.5, "z": 0},
                    "scale": {"x": 1000, "y": 1, "z": 1000}
                },
                "meshRenderer": {
                    "mesh": "primitives/plane",
                    "material": "materials/water.mat"
                }
            }
        },
        {
            "name": "DirectionalLight",
            "components": {
                "transform": {
                    "rotation": {"x": -45, "y": 30, "z": 0}
                },
                "light": {
                    "type": "directional",
                    "color": {"r": 255, "g": 240, "b": 200},
                    "intensity": 1.0
                }
            }
        },
        {
            "name": "AmbientLight",
            "components": {
                "light": {
                    "type": "ambient",
                    "color": {"r": 100, "g": 120, "b": 150},
                    "intensity": 0.3
                }
            }
        }
    ]
}
```

## Примеры целевых игр

### "Пиратские приливы" — что нужно

| Механика | Как реализовать |
|----------|-----------------|
| Корабль на волнах | `physics.Ship` + `physics.OceanWaves` |
| Брызги воды | `particles.NewWaterSplashEmitter` |
| Динамическая карта | `procgen.WorldMap.GenerateArchipelago` |
| AI пиратов | `ai.StateMachine` + `ai.NavGrid` |
| Бой на кораблях | `physics.Rigidbody` + коллизии |
| Туман на море | `shader.FogShader` |

### "Мечты нейросетей" — что нужно

| Механика | Как реализовать |
|----------|-----------------|
| Деформация пространства | `render.SpaceDeformer` |
| Психоделические цвета | `shader.PsychedelicShader` |
| Аудио-реактивность | `audio.AudioAnalyzer` + `AudioReactiveEffect` |
| Процедурный мир | `procgen.Noise2D.FBM` + `Heightmap` |
| Пульсация от музыки | `render.PulseDeform` + beat detection |
| Сюрреалистичные объекты | `render.TwistDeform`, `MeltDeform`, `NoiseDeform` |

## Советы по оптимизации

1. **Частицы**: не более 1000-2000 на сцену
2. **Pathfinding**: кэшируйте пути, обновляйте не каждый кадр
3. **Деформация**: применяйте к LOD мешам
4. **FFT**: используйте буфер 1024-2048 семплов
5. **Коллизии**: используйте broad-phase (spatial hash)

## Следующие шаги

1. Создайте проект: `kenga new mypiratgame`
2. Добавьте модели в `assets/models/`
3. Настройте сцену в `scenes/main.scene.json`
4. Напишите игровую логику в WASM или Go
5. Запустите: `kenga run`

---

Удачной разработки!
