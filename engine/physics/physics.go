package physics

import (
	emath "goenginekenga/engine/math"
)

// Rigidbody представляет физическое тело с массой и скоростью
type Rigidbody struct {
	Mass            float32    `json:"mass"`
	Velocity        emath.Vec3 `json:"velocity"`
	AngularVelocity emath.Vec3 `json:"angularVelocity"`
	Drag            float32    `json:"drag"`
	AngularDrag     float32    `json:"angularDrag"`
	UseGravity      bool       `json:"useGravity"`
	IsKinematic     bool       `json:"isKinematic"`
}

// DefaultRigidbody возвращает стандартный rigidbody
func DefaultRigidbody() *Rigidbody {
	return &Rigidbody{
		Mass:            1.0,
		Velocity:        emath.V3(0, 0, 0),
		AngularVelocity: emath.V3(0, 0, 0),
		Drag:            0.0,
		AngularDrag:     0.05,
		UseGravity:      true,
		IsKinematic:     false,
	}
}

// Collider представляет форму коллизии
type Collider struct {
	Type      string     `json:"type"` // box, sphere, capsule, mesh
	Center    emath.Vec3 `json:"center"`
	Size      emath.Vec3 `json:"size"`   // для box
	Radius    float32    `json:"radius"` // для sphere/capsule
	Height    float32    `json:"height"` // для capsule
	IsTrigger bool       `json:"isTrigger"`
}

// DefaultBoxCollider возвращает стандартный box collider
func DefaultBoxCollider() *Collider {
	return &Collider{
		Type:      "box",
		Center:    emath.V3(0, 0, 0),
		Size:      emath.V3(1, 1, 1),
		IsTrigger: false,
	}
}

// DefaultSphereCollider возвращает стандартный sphere collider
func DefaultSphereCollider() *Collider {
	return &Collider{
		Type:      "sphere",
		Center:    emath.V3(0, 0, 0),
		Radius:    0.5,
		IsTrigger: false,
	}
}

// DefaultCapsuleCollider возвращает стандартный capsule collider (вертикальная ось Y)
func DefaultCapsuleCollider() *Collider {
	return &Collider{
		Type:      "capsule",
		Center:    emath.V3(0, 0, 0),
		Radius:    0.5,
		Height:    2.0,
		IsTrigger: false,
	}
}

// PhysicsWorld управляет физической симуляцией
type PhysicsWorld struct {
	Gravity  emath.Vec3 `json:"gravity"`
	TimeStep float32    `json:"timeStep"`
	Substeps int        `json:"substeps"`
}

// DefaultPhysicsWorld возвращает стандартный физический мир
func DefaultPhysicsWorld() *PhysicsWorld {
	return &PhysicsWorld{
		Gravity:  emath.V3(0, -9.81, 0),
		TimeStep: 1.0 / 60.0,
		Substeps: 1,
	}
}

// Update выполняет шаг физической симуляции
func (pw *PhysicsWorld) Update(deltaTime float32, bodies []*Rigidbody, colliders []*Collider, transforms []emath.Vec3) {
	// Простая эйлерова интеграция
	for i, body := range bodies {
		if body.IsKinematic {
			continue
		}

		// Применяем гравитацию
		if body.UseGravity {
			gravityForce := pw.Gravity.Mul(body.Mass)
			body.Velocity = body.Velocity.Add(gravityForce.Mul(deltaTime))
		}

		// Применяем drag
		if body.Drag > 0 {
			dragForce := body.Velocity.Mul(-body.Drag)
			body.Velocity = body.Velocity.Add(dragForce.Mul(deltaTime))
		}

		// Обновляем позицию
		displacement := body.Velocity.Mul(deltaTime)
		transforms[i] = transforms[i].Add(displacement)

		// Применяем angular drag
		if body.AngularDrag > 0 {
			angularDrag := body.AngularVelocity.Mul(-body.AngularDrag)
			body.AngularVelocity = body.AngularVelocity.Add(angularDrag.Mul(deltaTime))
		}
	}

	// Простая коллизия с полом (y = 0)
	for i, body := range bodies {
		if transforms[i].Y < 0 {
			transforms[i].Y = 0
			if body.Velocity.Y < 0 {
				body.Velocity.Y = -body.Velocity.Y * 0.5 // bounce with energy loss
			}
		}
	}
}
