# Честный аудит: GoEngineKenga vs Unity vs Unreal Engine

Обновлённое сравнение по состоянию 2025. Без маркетинга.

---

## 1. Сводная таблица (актуальная)

| Критерий | GoEngineKenga | Unity | Unreal Engine 5 |
|----------|---------------|-------|-----------------|
| **Рендер 3D** | Software (Ebiten) ~5k tris + WebGPU PBR 10k+ tris | URP/HDRP, GPU | Lumen, Nanite, raytracing |
| **Материалы** | PBR (albedo, metallic, roughness, normal) | PBR, Shader Graph | PBR, node-based editor |
| **Освещение** | Ambient, directional, point | GI, baked, real-time | Lumen (real-time GI) |
| **Тени** | Directional shadow map (Ebiten + WebGPU) | Shadow maps | Cascaded shadow maps |
| **Постобработка** | Bloom, SSAO (Ebiten) | Bloom, SSAO, DOF | Полный набор |
| **LOD** | По дистанции (LODRefs), frustum culling | LOD groups | Nanite, HLOD |
| **Instancing** | GPU instancing (WebGPU), mesh cache | Да | Да |
| **Физика** | AABB, Sphere, Box, Capsule, Raycast, Distance/FixedPoint | PhysX, joints | Chaos |
| **Asset pipeline** | glTF: меши, материалы, текстуры, normal map | FBX, glTF | FBX, USD |
| **Анимация** | API (Clip, Animator, blend) — нет импорта skeletal из glTF | Mecanim, blend trees | Control Rig, Sequencer |
| **Редактор** | IDE (Tauri+Vue), viewport WebSocket, hot-reload | Полноценный | Blueprint, Sequencer |
| **Orbit camera** | ПКМ/СКМ/scroll в Ebiten | Да | Да |
| **Платформы** | Desktop, WASM, Mobile (gomobile) | Все | Все |
| **Сеть** | WebSocket API, Client/Server | Netcode | Replication |
| **Стоимость** | MIT, 0 | Подписка, runtime fee | Роялти после порога |

---

## 2. Где GoEngineKenga догоняет Unity/Unreal

| Область | Статус | Детали |
|---------|--------|--------|
| **PBR** | ✅ Есть | albedo, metallic, roughness, normal mapping |
| **Shadow map** | ✅ Есть | Directional, WebGPU + Ebiten |
| **Bloom, SSAO** | ✅ Есть | Software post-process |
| **LOD** | ✅ Есть | Mesh.LODRefs, переключение по дистанции |
| **Frustum culling** | ✅ Есть | ExtractFrustum, SphereInFrustum |
| **GPU instancing** | ✅ Есть | WebGPU, батчинг по mesh+material |
| **Mesh cache** | ✅ Есть | 10k+ tris без пересоздания буферов |
| **Joints** | ✅ Частично | DistanceConstraint, FixedPointConstraint |
| **Raycast** | ✅ Есть | physics.Raycast |
| **Orbit camera** | ✅ Есть | ПКМ/СКМ/scroll |
| **Hot-reload** | ✅ Есть | glTF, сцены, index |
| **WASM, Mobile** | ✅ Есть | Ebiten targets |

---

## 3. Где GoEngineKenga отстаёт

| Область | Разрыв | Unity/Unreal |
|---------|--------|--------------|
| **Масштаб сцен** | 10k tris vs миллионы | GPU, Nanite |
| **Skeletal animation** | API есть, импорт из glTF нет | Mecanim, Control Rig |
| **IBL, отражения** | Нет | Бaked/env probes |
| **DOF, motion blur** | Нет | Постобработка |
| **Occlusion culling** | Нет | Portal, PVS |
| **Point/spot shadows** | Только directional | Все типы |
| **Shader Graph** | Нет, фикс. шейдеры | Node-based |
| **Blueprint / визуальный скриптинг** | Нет | Blueprint, Bolt |
| **Asset Store** | Нет | Обширный |
| **Сообщество** | Минимальное | Миллионы |
| **Консоли** | Нет | Xbox, PlayStation |
| **VR/AR** | Нет | Поддержка |

---

## 4. Оценки по шкале 1–10

| Критерий | GoEngineKenga | Unity | Unreal |
|----------|---------------|-------|--------|
| 3D-графика (качество) | 5 | 8 | 10 |
| 3D-графика (производительность) | 5 (Ebiten) / 7 (WebGPU) | 8 | 10 |
| PBR/материалы | 7 | 8 | 10 |
| Тени | 6 | 8 | 9 |
| Физика | 6 | 8 | 8 |
| Анимация (skeletal) | 3 | 9 | 9 |
| Редактор/IDE | 5 | 9 | 9 |
| Платформы | 6 | 10 | 10 |
| Стоимость/открытость | 10 | 6 | 8 |
| Исходный код | 10 | 4 | 10 |
| Документация | 6 | 9 | 8 |
| Сообщество | 2 | 10 | 9 |
| Инди (подходит) | 8 | 10 | 6 |
| AAA (подходит) | 4 | 5 | 10 |
| CAD/viewport | 7 | 8 | 9 |

---

## 5. Рекомендуемые улучшения (приоритет)

1. **Импорт skeletal animation из glTF** — skin, joints, animation clips. Связь с Animator.
2. ~~**Инвалидация mesh cache при hot-reload**~~ — реализовано (frame.InvalidateMeshCache).
3. **Point/spot shadows** — для множественных источников.
4. **DOF (Depth of Field)** — простая постобработка для CAD/архитектуры.
5. **WebGPU: Orbit camera** — сейчас orbit только в Ebiten.
6. **Occlusion culling** — для больших сцен (опционально).

---

## 6. Вывод

GoEngineKenga — образовательный и прототипный движок с сильной базой для инди и CAD. PBR, тени, LOD, joints, orbit camera, hot-reload уже есть. Главные пробелы: skeletal animation из glTF, масштаб сцен (software ограничен), экосистема и сообщество.

Для инди-игр с простой 3D — годен (8/10). Для AAA — нет (4/10). Для CAD/viewport — приемлем (7/10).
