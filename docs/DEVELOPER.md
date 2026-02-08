# Руководство разработчика GoEngineKenga

Для тех, кто расширяет движок или встраивает его в свои проекты.

## Архитектура

- **Runtime** — игровой цикл, физика, коллизии, constraints. Режимы Edit/Play.
- **ECS** — entities, компоненты (Transform, MeshRenderer, Rigidbody, Collider, Camera, Light).
- **Render** — Ebiten (software rasterizer) или WebGPU. PBR, shadow map, bloom, SSAO.
- **Asset** — Resolver для мешей, материалов, текстур. Hot-reload через fsnotify.
- **IDE** — Tauri + Vue, WebSocket для live viewport.

## Физика и constraints

### DistanceConstraint

Связывает два тела «палкой» фиксированной длины:

```go
rt.Constraints = append(rt.Constraints, &physics.DistanceConstraint{
    EntityA:    idA,
    EntityB:    idB,
    RestLength: 3.0, // 0 = вычислить из текущих позиций
})
```

### FixedPointConstraint

Привязывает тело к мировой точке:

```go
rt.Constraints = append(rt.Constraints, &physics.FixedPointConstraint{
    EntityID: id,
    Point:    emath.V3(0, 5, 0),
})
```

### Добавление своих constraints

Реализуйте интерфейс `physics.Constraint`:

```go
type MyConstraint struct { ... }

func (c *MyConstraint) Apply(positions map[physics.EntityID]*emath.Vec3, rigidbodies map[physics.EntityID]*physics.Rigidbody, iterations int) {
    // Коррекция позиций и скоростей
}
```

## Сборка

```bash
# Ebiten (по умолчанию)
go build -o kenga.exe ./cmd/kenga

# WebGPU (CGO)
go build -tags webgpu -o kenga.exe ./cmd/kenga

# WASM
./scripts/build-wasm.ps1
```

## Расширение рендерера

- **Ebiten** — `engine/render/ebiten/renderer3d.go`, `rasterizer.go`
- **WebGPU** — `engine/render/webgpu/`, WGSL-шейдеры в `shader.wgsl`, `shadow.wgsl`
- **Mesh cache** — vertex buffers кэшируются по meshAssetID. При hot-reload (ConsumeAssetsDirty/IndexDirty) кэш инвалидируется (frame.InvalidateMeshCache).
- Постобработка: `render.ApplyBloom`, `render.ApplySSAO`

## Orbit camera (Ebiten)

В runtime-окне: ПКМ — вращение вокруг target, СКМ — панорама, колёсико — зум. Состояние в `render.OrbitState`. Сброс при hot-reload сцены.

## IDE (Tauri)

- Frontend: `ide/` (Vue + Vite)
- Rust: `ide/src-tauri/`
- Команды: `invoke('command_name', { ... })`
- Состояние: `app.manage(MyState{})`, `State<MyState>` в командах

## Референсы

- [Ebiten](https://github.com/hajimehoshi/ebiten) — рендер, input
- [Tauri v2](https://v2.tauri.app/) — IDE
- [Jolt Physics](https://github.com/jrouwe/JoltPhysics) — референс по constraints (Fixed, Distance, Hinge и т.д.)
- [docs/REFERENCES.md](REFERENCES.md) — ссылки на glTF, WebGPU, профилирование
