package physics

import (
	"math"
	"testing"

	emath "goenginekenga/engine/math"
)

// step прогоняет n шагов физики с dt=1/60 и возвращает обновлённые позиции
// (physics.Update мутирует элементы среза, а не переменные вызывающего).
func step(pw *PhysicsWorld, bodies []*Rigidbody, transforms []emath.Vec3, n int) []emath.Vec3 {
	for i := 0; i < n; i++ {
		pw.Update(1.0/60.0, bodies, nil, transforms)
	}
	return transforms
}

func TestGravityIntegration(t *testing.T) {
	pw := DefaultPhysicsWorld()
	body := DefaultRigidbody()
	body.UseGravity = true

	pos := step(pw, []*Rigidbody{body}, []emath.Vec3{emath.V3(0, 10, 0)}, 60) // 1 секунда

	// v = -9.81 м/с; полунеявный Эйлер (гравитация до смещения): y = 10 - 9.81/3600*Σ(1..60) ≈ 5.01
	if math.Abs(float64(body.Velocity.Y)+9.81) > 0.01 {
		t.Fatalf("velocity.Y=%v, want ≈-9.81", body.Velocity.Y)
	}
	if math.Abs(float64(pos[0].Y)-5.01) > 0.1 {
		t.Fatalf("position.Y=%v, want ≈5.01", pos[0].Y)
	}
}

func TestFloorCollisionBounce(t *testing.T) {
	pw := DefaultPhysicsWorld()
	body := DefaultRigidbody()
	body.UseGravity = false
	body.Velocity = emath.V3(0, -5, 0)

	pos := step(pw, []*Rigidbody{body}, []emath.Vec3{emath.V3(0, -0.5, 0)}, 1)

	// Пол на y=0: позиция прижата к нулю, скорость отскочила с потерей энергии (×0.5)
	if pos[0].Y != 0 {
		t.Fatalf("position.Y=%v, want 0 (floor)", pos[0].Y)
	}
	if body.Velocity.Y != 2.5 {
		t.Fatalf("velocity.Y=%v, want 2.5 (bounce)", body.Velocity.Y)
	}
}

func TestKinematicBodyUnmoved(t *testing.T) {
	pw := DefaultPhysicsWorld()
	body := DefaultRigidbody()
	body.IsKinematic = true
	body.UseGravity = true

	pos := step(pw, []*Rigidbody{body}, []emath.Vec3{emath.V3(1, 2, 3)}, 120) // 2 секунды

	if pos[0].X != 1 || pos[0].Y != 2 || pos[0].Z != 3 {
		t.Fatalf("kinematic body moved: %v", pos[0])
	}
	if body.Velocity != (emath.Vec3{}) {
		t.Fatalf("kinematic body got velocity: %v", body.Velocity)
	}
}

func TestDragDecaysVelocity(t *testing.T) {
	pw := DefaultPhysicsWorld()
	body := DefaultRigidbody()
	body.UseGravity = false
	body.Drag = 1.0 // e^(-1) ≈ 0.368 за секунду
	body.Velocity = emath.V3(10, 0, 0)

	step(pw, []*Rigidbody{body}, []emath.Vec3{emath.V3(0, 0, 0)}, 60) // 1 секунда

	// Эйлер: 10*(1 - dt)^60 ≈ 3.64; ожидаем 3.4..3.9
	if body.Velocity.X < 3.4 || body.Velocity.X > 3.9 {
		t.Fatalf("velocity.X=%v, want ≈3.6", body.Velocity.X)
	}
}
