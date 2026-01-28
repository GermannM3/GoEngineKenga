package runtime

import (
	"image/color"
	"time"

	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/physics"
)

// Step продвигает время и возвращает dt.
func (rt *Runtime) Step() time.Duration {
	now := time.Now()
	dt := now.Sub(rt.lastTick)
	if dt <= 0 {
		dt = time.Second / 60
	}
	rt.lastTick = now

	if rt.Mode == ModePlay && rt.PlayWorld != nil {
		if rt.Profiler != nil {
			timer := rt.Profiler.StartSection("physics")
			rt.stepPhysics(float32(dt.Seconds()))
			timer.End()
			rt.Profiler.SetEntityCount(len(rt.PlayWorld.Entities()))
		} else {
			rt.stepPhysics(float32(dt.Seconds()))
		}
	}

	return dt
}

// stepPhysics выполняет шаг физической симуляции
func (rt *Runtime) stepPhysics(deltaTime float32) {
	if rt.PlayWorld == nil {
		return
	}

	// Собираем все rigidbody и их позиции
	entityIDs := []ecs.EntityID{}
	rigidbodies := make(map[ecs.EntityID]*physics.Rigidbody)
	positions := make(map[ecs.EntityID]*emath.Vec3)

	for _, id := range rt.PlayWorld.Entities() {
		if rb, ok := rt.PlayWorld.GetRigidbody(id); ok {
			entityIDs = append(entityIDs, id)
			rbCopy := rb // Make a copy to get pointer
			rigidbodies[id] = &rbCopy

			if tr, ok := rt.PlayWorld.GetTransform(id); ok {
				pos := tr.Position
				positions[id] = &pos
			} else {
				pos := emath.V3(0, 0, 0)
				positions[id] = &pos
			}
		}
	}

	// Apply physics (gravity, velocity integration)
	if len(entityIDs) > 0 {
		// Build slices for PhysicsWorld.Update
		var bodies []*physics.Rigidbody
		var posSlice []emath.Vec3
		for _, id := range entityIDs {
			bodies = append(bodies, rigidbodies[id])
			posSlice = append(posSlice, *positions[id])
		}

		rt.PhysicsWorld.Update(deltaTime, bodies, nil, posSlice)

		// Copy positions back
		for i, id := range entityIDs {
			*positions[id] = posSlice[i]
		}
	}

	// Detect and resolve collisions
	if rt.CollisionManager != nil {
		// Build collider data
		var colliders []physics.ColliderData
		for _, id := range rt.PlayWorld.Entities() {
			if col, ok := rt.PlayWorld.GetCollider(id); ok {
				pos := emath.Vec3{}
				if p, ok := positions[id]; ok && p != nil {
					pos = *p
				} else if tr, ok := rt.PlayWorld.GetTransform(id); ok {
					pos = tr.Position
				}
				colCopy := col
				colliders = append(colliders, physics.ColliderData{
					ID:       uint64(id),
					Collider: &colCopy,
					Position: pos,
				})
			}
		}

		// Convert maps to use physics.EntityID (uint64)
		physPositions := make(map[physics.EntityID]*emath.Vec3)
		physRigidbodies := make(map[physics.EntityID]*physics.Rigidbody)
		for id, pos := range positions {
			physPositions[uint64(id)] = pos
		}
		for id, rb := range rigidbodies {
			physRigidbodies[uint64(id)] = rb
		}

		rt.CollisionManager.DetectAndResolve(colliders, physPositions, physRigidbodies)
	}

	// Write positions back to world
	for id, pos := range positions {
		if tr, ok := rt.PlayWorld.GetTransform(id); ok {
			tr.Position = *pos
			rt.PlayWorld.SetTransform(id, tr)
		}
		// Also update rigidbody velocity
		if rb, ok := rigidbodies[id]; ok {
			rt.PlayWorld.SetRigidbody(id, *rb)
		}
	}

	// Дополнительный шаг: обновляем системы нанесения мастики (dispensers).
	rt.stepDispensers(deltaTime)
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

// stepDispensers добавляет точки траектории для активных Dispenser-компонентов.
func (rt *Runtime) stepDispensers(deltaTime float32) {
	if rt.PlayWorld == nil {
		return
	}

	w := rt.PlayWorld
	for _, id := range w.Entities() {
		disp, ok := w.GetDispenser(id)
		if !ok || !disp.Active {
			continue
		}

		tr, ok := w.GetTransform(id)
		if !ok {
			continue
		}

		pos := tr.Position
		if !disp.HasLast {
			disp.LastPosition = pos
			disp.HasLast = true
			w.SetDispenser(id, disp)
		}

		// Минимальное расстояние между точками зависит от FlowRate:
		// чем выше расход, тем чаще точки.
		minStep := float32(0.01)
		if disp.FlowRate > 0 {
			minStep = 0.05 / disp.FlowRate
		}
		if minStep < 0.005 {
			minStep = 0.005
		}

		if pos.Sub(disp.LastPosition).Len() < minStep {
			continue
		}

		traj, _ := w.GetTrajectory(id)
		traj.Points = append(traj.Points, pos)

		if traj.Color.A == 0 {
			if disp.Color.A != 0 {
				traj.Color = disp.Color
			} else {
				traj.Color = color.RGBA{R: 200, G: 220, B: 255, A: 255}
			}
		}
		if traj.Width <= 0 {
			if disp.Radius > 0 {
				traj.Width = disp.Radius * 2
			} else {
				traj.Width = 3
			}
		}

		w.SetTrajectory(id, traj)

		disp.LastPosition = pos
		w.SetDispenser(id, disp)
	}
}
