# Аудит и дорожная карта: GoEngineKenga → AAA-уровень

Честное сравнение с Unreal Engine 5 и Unity. План доработки по блокам до коммерческой готовности.

**Движок используется в двух сценариях:** игры (runtime, физика, анимация) и CAD-подобные приложения (KengaCAD, рендер моделей, viewport). Hot-reload ассетов и сцен критичен для обоих.

---

## 1. Сводная таблица сравнения

| Критерий | GoEngineKenga | Unity | Unreal Engine 5 |
|----------|---------------|-------|----------------|
| **Рендер** | Software (Ebiten) + WebGPU PBR, mesh cache | URP/HDRP, GPU | Lumen, Nanite, raytracing |
| **Материалы** | PBR (albedo, metallic, roughness, normal) | PBR, Shader Graph | PBR, node-based editor |
| **Освещение** | Ambient, directional, point | GI, baked | Lumen (real-time GI) |
| **Тени** | Shadow map (directional) | Shadow maps | Cascaded shadow maps |
| **Постобработка** | Bloom, SSAO | Bloom, SSAO, DOF | Полный набор |
| **LOD** | LODRefs, frustum culling | LOD groups | Nanite, HLOD |
| **Физика** | AABB, Sphere, Box, Capsule, Raycast, joints | PhysX, joints | Chaos |
| **Анимация** | Skeletal, keyframe, sprite | Mecanim, blend trees | Control Rig, sequencer |
| **Редактор** | IDE (Tauri), viewport WebSocket, orbit camera | Полноценный | Blueprint, Sequencer |
| **Платформы** | Desktop, WASM, Mobile | Все | Все |
| **Сеть** | Client/Server API | Netcode | Replication |
| **Asset pipeline** | glTF, JSON | FBX, много форматов | FBX, USD |

---

## 2. Критические пробелы (блокируют AAA)

### 2.1 Масштаб сцен

**Текущее:** Ebiten ~5k tris, WebGPU 10k+ tris с mesh cache. Instancing, LOD, frustum culling есть.

**UE5/Unity:** Миллионы полигонов, Nanite, Lumen.

**Разрыв:** Software rasterizer ограничен. WebGPU требует CGO. Для AAA нужен больший масштаб.

### 2.2 Skeletal animation

**Текущее:** API (Clip, Animator, blend) есть. Импорт skin/animation из glTF — реализован (Skin, JOINTS_0, WEIGHTS_0, inverse bind matrices, Animation channels → Clip).

**Осталось:** Связь с рендерером (skinning в vertex shader), привязка к ECS.

### 2.3 IBL, отражения

**Текущее:** Нет baked environment, отражений.

**Требуется:** Опционально для PBR-качества.

### 2.4 Occlusion culling

**Текущее:** Нет.

**Требуется:** Portal, PVS или Hi-Z для больших сцен.

---

## 3. Важные пробелы (снижают конкурентоспособность)

- **Point/spot shadows:** Только directional shadow map
- **DOF, motion blur:** Нет
- **Shader Graph:** Нет, только фиксированные шейдеры
- ~~**Mesh cache invalidation**~~ — реализовано
- **WebGPU orbit camera:** Orbit только в Ebiten

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
- [x] 3.3 SSAO (screen-space ambient occlusion, software rasterizer)

### Блок 4: LOD и оптимизация (приоритет 4) ✅
- [x] 4.1 LOD levels в asset (Mesh.LODRefs, переключение по дистанции)
- [x] 4.2 Frustum culling
- [x] 4.3 GPU instancing (одинаковые меши)

### Блок 5: Физика (приоритет 5) ✅
- [x] 5.1 Capsule collider (DefaultCapsuleCollider, CheckCapsuleCapsule/Sphere/Box)
- [x] 5.2 Raycast API (physics.Raycast, RaycastHit)
- [x] 5.3 Joints/constraints (DistanceConstraint, FixedPointConstraint, ResolveConstraints)

### Блок 6: Редактор (приоритет 6)
- [x] 6.1 IDE Viewport: real-time preview через WebSocket
- [x] 6.2 Drag-and-drop ассетов (glTF на окно → assets → import)
- [x] 6.3 Asset/Scene hot-reload (fsnotify: glTF, сцены, index → автообновление viewport)
- [x] 6.4 Orbit camera: ПКМ rotate, СКМ pan, scroll zoom (Ebiten)

### Блок 7: Платформы (приоритет 7)
- [x] 7.1 WebAssembly (Ebiten на WASM)
- [x] 7.2 Mobile (Ebiten ebitenmobile: Android .aar, iOS .xcframework)

---

## 5. Порядок реализации

1. **Блок 1** — без GPU двигатель не конкурентен. Это первый шаг.
2. **Блок 2** — PBR даёт визуальный уровень ближе к индустрии.
3. **Блок 3** — тени и bloom заметно улучшают картинку.
4. **Блок 4** — LOD и culling для больших сцен.
5. **Блоки 5–7** — по мере необходимости.

---

## 6. Критерии готовности к коммерческому использованию

- [x] GPU-рендер сцен 10k+ треугольников при 60 FPS (WebGPU: instancing + mesh cache)
- [x] PBR материалы
- [x] Тени (directional shadow map)
- [x] Стабильный редактор/IDE (window-state: сохранение позиции/размера)
- [x] Документация для разработчиков ([docs/DEVELOPER.md](DEVELOPER.md))
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

Артефакты: Windows, Linux, macOS (amd64/arm64), WASM. Подробнее: [docs/RELEASE.md](RELEASE.md).

Платформы: WASM ([wasm/README.md](wasm/README.md)), Mobile ([docs/MOBILE.md](MOBILE.md)).

---

## 9. Честный аудит

Подробное сравнение с Unity и Unreal: [docs/AUDIT_UNITY_UNREAL.md](AUDIT_UNITY_UNREAL.md).

---

*Документ обновлён: 2025*
