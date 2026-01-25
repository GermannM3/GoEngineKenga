package runtime

import (
	"time"

	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
)

// Step продвигает время и возвращает dt.
func (rt *Runtime) Step() time.Duration {
	now := time.Now()
	dt := now.Sub(rt.lastTick)
	if dt <= 0 {
		dt = time.Second / 60
	}
	rt.lastTick = now

	// Выполняем физическую симуляцию
	if rt.Mode == ModePlay && rt.PlayWorld != nil {
		rt.stepPhysics(float32(dt.Seconds()))
	}

	return dt
}

// stepPhysics выполняет шаг физической симуляции
func (rt *Runtime) stepPhysics(deltaTime float32) {
	if rt.PlayWorld == nil {
		return
	}

	// Собираем все rigidbody и их позиции
	var bodies []*ecs.Rigidbody
	var positions []emath.Vec3

	for _, id := range rt.PlayWorld.Entities() {
		if rb, ok := rt.PlayWorld.GetRigidbody(id); ok {
			bodies = append(bodies, &rb)
			if tr, ok := rt.PlayWorld.GetTransform(id); ok {
				positions = append(positions, tr.Position)
			} else {
				positions = append(positions, emath.V3(0, 0, 0))
			}
		}
	}

	if len(bodies) > 0 {
		rt.PhysicsWorld.Update(deltaTime, bodies, nil, positions)

		// Обновляем позиции в мире
		bodyIndex := 0
		for _, id := range rt.PlayWorld.Entities() {
			if _, ok := rt.PlayWorld.GetRigidbody(id); ok {
				if tr, ok := rt.PlayWorld.GetTransform(id); ok {
					tr.Position = positions[bodyIndex]
					rt.PlayWorld.SetTransform(id, tr)
				}
				bodyIndex++
			}
		}
	}
}

// v0: простая «система», чтобы видеть, что PlayWorld реально живёт отдельно.
func SpinSystem(w *ecs.World, dt time.Duration) {
	ids := w.Entities()
	for _, id := range ids {
		t, ok := w.GetTransform(id)
		if !ok {
			continue
		}
		// вращаем вокруг Y
		t.Rotation = t.Rotation.Add(emath.V3(0, float32(dt.Seconds()*30.0), 0))
		w.SetTransform(id, t)
	}
}

