package runtime

import (
	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/physics"
)

// CharacterController — кинематическое управление персонажем (как в Unity):
// горизонтальная скорость задаётся вводом, гравитация — физикой,
// а Grounded вычисляется здесь коротким raycast'ом вниз от низа коллайдера.
// Вызывается в конце физического шага (после интеграции и разрешения коллизий),
// чтобы ввод следующего кадра видел актуальное состояние «на земле».

const (
	groundSkin     = 0.3 // глубина «прилипания» к земле под низом коллайдера
	groundRayStart = 0.05 // луч стартует чуть выше низа, чтобы не попасть внутрь себя
)

// UpdateCharacterControllers обновляет Grounded всех CharacterController в мире
// и гасит вертикальную скорость на земле (убирает дребезг от отскока пола).
func UpdateCharacterControllers(w *ecs.World) {
	if w == nil {
		return
	}

	// Собираем коллайдеры мира один раз для raycast'ов
	colliders := make([]physics.ColliderData, 0, 64)
	positions := make(map[physics.EntityID]emath.Vec3, 64)
	for _, id := range w.Entities() {
		col, ok := w.GetCollider(id)
		if !ok {
			continue
		}
		tr, hasTr := w.GetTransform(id)
		if !hasTr {
			continue
		}
		colliders = append(colliders, physics.ColliderData{ID: uint64(id), Collider: &col, Position: tr.Position})
		positions[uint64(id)] = tr.Position
	}
	if len(colliders) == 0 {
		return
	}

	for _, id := range w.Entities() {
		cc, ok := w.GetCharacterController(id)
		if !ok {
			continue
		}
		col, hasCol := w.GetCollider(id)
		tr, hasTr := w.GetTransform(id)
		if !hasCol || !hasTr {
			cc.Grounded = false
			w.SetCharacterController(id, cc)
			continue
		}

		// Низ коллайдера в мировых координатах
		bottom := tr.Position.Y + col.Center.Y - colliderHalfHeight(&col)
		origin := emath.Vec3{X: tr.Position.X, Y: bottom + groundRayStart, Z: tr.Position.Z}

		// Луч вниз, исключая собственный коллайдер
		others := make([]physics.ColliderData, 0, len(colliders))
		for _, c := range colliders {
			if c.ID != uint64(id) {
				others = append(others, c)
			}
		}
		hit := physics.Raycast(origin, emath.Vec3{X: 0, Y: -1, Z: 0}, groundSkin+groundRayStart, others, positions)

		cc.Grounded = hit != nil
		if cc.Grounded {
			if rb, hasRb := w.GetRigidbody(id); hasRb && rb.Velocity.Y < 0 {
				rb.Velocity.Y = 0
				w.SetRigidbody(id, rb)
			}
		}
		w.SetCharacterController(id, cc)
	}
}

// colliderHalfHeight — расстояние от центра коллайдера до его низа.
func colliderHalfHeight(c *physics.Collider) float32 {
	switch c.Type {
	case "capsule":
		return c.Height/2 + c.Radius
	case "sphere":
		return c.Radius
	default: // box
		return c.Size.Y / 2
	}
}
