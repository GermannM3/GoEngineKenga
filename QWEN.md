# GoEngineKenga Project Context

## Project Overview

GoEngineKenga is a comprehensive, standalone game engine built in Go. It's designed to be a full-featured alternative to commercial engines like Unity and Unreal Engine, with the goal of surpassing them in capabilities while maintaining zero licensing costs. The engine is completely autonomous and doesn't require paid services or external dependencies.

### Key Features

**Engine Core:**
- ECS (Entity-Component-System) architecture
- Components: Transform, Camera, MeshRenderer, Rigidbody, Collider, Light, AudioSource, UICanvas
- Scenes and prefabs in JSON format
- Physics: gravity, object-object collisions (AABB, Sphere, Box-Sphere), impulses
- Input: keyboard, mouse (IsKeyPressed, IsMouseButtonPressed, delta, scroll)
- Audio: WAV/MP3/OGG, 3D spatial audio, FFT analysis
- UI: Button, Label, Panel with automatic hover/click

**3D Graphics:**
- Software 3D renderer: rasterization with Z-buffer
- Textures: support for textured models
- Lighting: Ambient, Directional, Point lights
- Camera: perspective, LookAt, FOV
- Meshes: cube, sphere, plane, cylinder + glTF import

**Advanced Systems:**
- Particle systems with presets (fire, smoke, explosion, water splashes, sparks)
- Animation: skeletal, keyframe, sprite
- Procedural generation: Perlin noise, dungeon generation, world maps
- AI for NPCs: pathfinding, behavior trees, state machines
- Water physics: wave simulation, Gerstner waves, buoyancy
- Shaders and effects: toon, psychedelic, glitch, fog, post-processing
- Space deformation: wave, twist, bend, pulse, noise, sphere attract, melt
- Audio-reactivity: FFT analysis, beat detection, visualizers

**Development Tools:**
- CLI: `kenga new`, `kenga run`, `kenga import`, `kenga script build`
- WASM scripting: TinyGo → WASM with hot reload
- Asset pipeline: automatic import to `.kenga/`
- Plugins: extensible architecture

## Building and Running

### Prerequisites
- Go 1.22+
- TinyGo (optional, for WASM scripts)

### Installation Options

**Option 1 — Clone and build locally:**
```bash
git clone https://github.com/GermannM3/GoEngineKenga.git
cd GoEngineKenga
go mod tidy
go build -o kenga ./cmd/kenga
```

**Option 2 — Install pre-built binary:**
- Windows: run installer from releases or build single exe: `scripts\make-setup.bat`
- Linux: `./scripts/install-linux.sh` or .deb package

### Creating a New Project
```bash
# Create project (templates: default, platformer, topdown)
go run ./cmd/kenga new mygame --template platformer
cd mygame

# Run game
go run ../cmd/kenga run --project . --scene scenes/main.scene.json --backend ebiten
```

### Running Examples
```bash
# Import assets
go run ./cmd/kenga import --project samples/hello

# Run
go run ./cmd/kenga run --project samples/hello --scene scenes/main.scene.json --backend ebiten
```

If the `kenga` binary is installed, you can call it directly:
```bash
kenga run --project samples/hello --scene scenes/main.scene.json --backend ebiten
```

## Project Structure

```
engine/
├── ai/           # AI, pathfinding, behavior trees
├── animation/    # Animation (skeletal, sprite)
├── api/          # API definitions
├── asset/        # Asset pipeline
├── audio/        # Audio, FFT analysis
├── cli/          # CLI tools
├── convert/      # Format conversion utilities
├── debug/        # Debugging tools
├── ecs/          # Entity-Component-System
├── gameplay/     # Gameplay systems
├── input/        # Input (keyboard, mouse)
├── math/         # Math utilities
├── network/      # Network functionality
├── particles/    # Particle systems
├── physics/      # Physics, collisions, water
├── plugin/       # Plugin architecture
├── procgen/      # Procedural generation
├── profiler/     # Profiling tools
├── project/      # Project management
├── render/       # Graphics (3D render, camera, meshes)
├── runtime/      # Runtime systems
├── scene/        # Scene loading
├── script/       # WASM runtime
├── shader/       # Programmable shaders
└── ui/           # UI elements
```

## Development Conventions

### Coding Style
- Follow Go best practices and idioms
- Use clear, descriptive variable and function names
- Include comprehensive comments for exported functions
- Maintain clean, readable code structure

### Testing
- Write unit tests for new functionality
- Follow table-driven test patterns where appropriate
- Ensure all tests pass before committing

### Architecture
- Use the ECS (Entity-Component-System) pattern for game objects
- Keep components focused and single-purpose
- Separate engine logic from game-specific code
- Maintain clean interfaces between subsystems

### CLI Commands
| Command | Description |
|---------|-------------|
| `kenga new <name>` | Create project |
| `kenga run --project <path>` | Run game |
| `kenga import --project <path>` | Import assets |
| `kenga script build --project <path>` | Build WASM scripts |

## WebSocket Integration for CAD/CAM Applications

GoEngineKenga can be used as an embedded 3D visualization engine for desktop applications (Python/PyQt, Qt/QML, .NET, etc.) through a WebSocket API. This enables integration with CAD/CAM/robotics applications where the engine runs as a separate process and receives commands via WebSocket.

Key commands include:
- `load_model` - Load 3D models
- `clear_scene` - Clear the scene
- `set_camera` - Control camera position and properties
- `set_transform` - Set entity positions, rotations, scales
- `set_trajectory` - Draw trajectory lines
- `set_joint` - Control robot joints
- `start_dispensing`/`stop_dispensing` - Control dispensing effects (for robotics applications)

## Mission Statement

The project aims to become the ultimate game engine, surpassing both Unity (for indie developers) and Unreal Engine (for AAA projects). Key advantages include zero licensing costs (MIT license), clean modern Go codebase, full source code access, native performance, and no vendor lock-in.

## Development Roadmap

The project has ambitious goals to reach 10/10 ratings in all categories including 3D graphics, physics, audio, AI, UI, visual editor, asset pipeline, networking, platform support, performance, documentation, and community.

## License

MIT License - completely free to use and modify.