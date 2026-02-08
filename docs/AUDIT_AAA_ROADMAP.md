# Аудит и дорожная карта: GoEngineKenga → AAA-уровень

Честное сравнение с Unreal Engine 5 и Unity. План доработки по блокам до коммерческой готовности.

**Движок используется в двух сценариях:** игры (runtime, физика, анимация) и CAD-подобные приложения (KengaCAD, рендер моделей, viewport). Hot-reload ассетов и сцен критичен для обоих.

---

## 1. Сводная таблица сравнения

| Критерий | GoEngineKenga | Unity | Unreal Engine 5 |
|----------|---------------|-------|----------------|
| **Рендер** | Software (Ebiten) + WebGPU PBR | URP/HDRP, GPU | Lumen, Nanite, raytracing |
| **Материалы** | PBR (albedo, metallic, roughness) | PBR, Shader Graph | PBR, node-based editor |
| **Освещение** | Ambient, directional, point | GI, shadows, baked | Lumen (real-time GI), shadows |
| **Тени** | Shadow map (directional) | Shadow maps | Cascaded shadow maps |
| **Постобработка** | Vignette, chromatic aberration | Bloom, SSAO, DOF | Полный набор |
| **LOD** | Quality presets (draw calls) | LOD groups | Nanite, HLOD |
| **Физика** | AABB, Sphere, Box-Sphere, импульсы | PhysX, joints | Chaos |
| **Анимация** | Skeletal, keyframe, sprite | Mecanim, blend trees | Control Rig, sequencer |
| **Редактор** | IDE (Tauri), Web (Vue) | Полноценный | Blueprint, Sequencer |
| **Платформы** | Desktop (Ebiten) | Все | Все |
| **Сеть** | Client/Server API | Netcode | Replication |
| **Asset pipeline** | glTF, JSON | FBX, много форматов | FBX, USD |

---

## 2. Критические пробелы (блокируют AAA)

### 2.1 GPU-рендеринг — главный bottleneck

**Текущее состояние:** Software rasterizer, ~1000–5000 треугольников при 60 FPS. WebGPU backend только рисует один треугольник.

**UE5/Unity:** GPU, миллионы полигонов, instancing.

**Требуется:**
- Полноценный WebGPU backend с рендером ECS-сцены
- Или OpenGL/Vulkan backend
- Vertex/index buffers, MVP матрицы, текстуры

### 2.2 PBR и освещение

**Текущее:** Lambert diffuse, ambient, directional, point.

**Требуется:** Albedo, metallic, roughness, normal maps. Fresnel, IBL (опционально).

### 2.3 Тени

**Текущее:** Нет.

**Требуется:** Shadow maps (directional, point).

### 2.4 Физика

**Текущее:** Spatial hash, AABB/Sphere/Box коллизии, импульсы. Нет джойнтов, рэгдолла.

**Требуется:** Интеграция с Bullet/Jolt или расширение до constraints, joints.

---

## 3. Важные пробелы (снижают конкурентоспособность)

- **LOD:** Quality presets есть, но нет переключения мешей по дистанции
- **Frustum culling:** Проверить наличие
- **Occlusion culling:** Нет
- **Постобработка:** Частично (vignette, outline)
- **Редактор:** IDE отделён от рантайма, нет real-time preview
- **Документация:** Базовый уровень

---

## 4. План доработки по блокам

### Блок 1: GPU рендеринг (приоритет 1) ✅
**Цель:** WebGPU backend рендерит полную 3D-сцену из ECS.

- [x] 1.1 Game loop в WebGPU: Frame, OnUpdate, Resolver
- [x] 1.2 WGSL-шейдер: vertex (MVP) + fragment (Lambert lighting)
- [x] 1.3 Upload mesh: vertex buffer (pos, normal, uv), index buffer
- [x] 1.4 Рендер всех MeshRenderer из World
- [x] 1.5 Камера из ECS Camera
- [x] 1.6 Текстуры через asset.Resolver (MaterialID в mesh, BaseColorTex в material)

### Блок 2: PBR материалы (приоритет 2) ✅
- [x] 2.1 PBR shader: albedo, metallic, roughness
- [x] 2.2 Normal mapping (TBN, sample perturbed normal в software rasterizer)
- [x] 2.3 Directional lights в PBR (из ECS)

### Блок 3: Тени и постобработка (приоритет 3) ✅
- [x] 3.1 Shadow map (directional light)
- [x] 3.2 Bloom (Ebiten software rasterizer)
- [ ] 3.3 SSAO (опционально)

### Блок 4: LOD и оптимизация (приоритет 4) ✅
- [x] 4.1 LOD levels в asset (Mesh.LODRefs, переключение по дистанции)
- [x] 4.2 Frustum culling
- [x] 4.3 GPU instancing (одинаковые меши)

### Блок 5: Физика (приоритет 5) ✅
- [x] 5.1 Capsule collider (DefaultCapsuleCollider, CheckCapsuleCapsule/Sphere/Box)
- [x] 5.2 Raycast API (physics.Raycast, RaycastHit)
- [ ] 5.3 Joints/constraints (опционально)

### Блок 6: Редактор (приоритет 6)
- [x] 6.1 IDE Viewport: real-time preview через WebSocket
- [x] 6.2 Drag-and-drop ассетов (glTF на окно → assets → import)
- [x] 6.3 Asset/Scene hot-reload (fsnotify: glTF, сцены, index → автообновление viewport)

### Блок 7: Платформы (приоритет 7)
- [ ] 7.1 WebAssembly (Ebiten на WASM)
- [ ] 7.2 Mobile (Ebiten gomobile)

---

## 5. Порядок реализации

1. **Блок 1** — без GPU двигатель не конкурентен. Это первый шаг.
2. **Блок 2** — PBR даёт визуальный уровень ближе к индустрии.
3. **Блок 3** — тени и bloom заметно улучшают картинку.
4. **Блок 4** — LOD и culling для больших сцен.
5. **Блоки 5–7** — по мере необходимости.

---

## 6. Критерии готовности к коммерческому использованию

- [ ] GPU-рендер сцен 10k+ треугольников при 60 FPS
- [ ] PBR материалы
- [ ] Тени (хотя бы directional)
- [ ] Стабильный редактор/IDE
- [ ] Документация для разработчиков
- [ ] Один опубликованный инди-проект на движке

---

## 7. Сборка WebGPU

```bash
go build -tags webgpu -o kenga.exe ./cmd/kenga
```

Требования: CGO, wgpuglfw (могут быть ограничения по платформам). Без тега используется Ebiten (software rasterizer).

---

## 8. GitHub Release

Релизы создаются при push тега `v*`:

```bash
git tag v0.2.0
git push origin v0.2.0
```

Артефакты: Windows, Linux, macOS (amd64/arm64). Подробнее: [docs/RELEASE.md](RELEASE.md).

---

*Документ обновлён: 2025*
