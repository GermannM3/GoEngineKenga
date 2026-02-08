package physics

import (
	"math"

	emath "goenginekenga/engine/math"
)

// Constraint — интерфейс для ограничения, связывающего тела
type Constraint interface {
	// Apply применяет ограничение к позициям и скоростям тел
	Apply(positions map[EntityID]*emath.Vec3, rigidbodies map[EntityID]*Rigidbody, iterations int)
}

// DistanceConstraint — «палка» между двумя телами, сохраняет расстояние
type DistanceConstraint struct {
	EntityA    EntityID
	EntityB    EntityID
	RestLength float32   // целевое расстояние (0 = вычислить из начальных позиций)
	AnchorA    emath.Vec3 // локальная точка на A (пока используем центр)
	AnchorB    emath.Vec3 // локальная точка на B
}

// Apply реализует Constraint
func (c *DistanceConstraint) Apply(positions map[EntityID]*emath.Vec3, rigidbodies map[EntityID]*Rigidbody, iterations int) {
	posA := positions[c.EntityA]
	posB := positions[c.EntityB]
	if posA == nil || posB == nil {
		return
	}

	rbA := rigidbodies[c.EntityA]
	rbB := rigidbodies[c.EntityB]

	// World anchors (с учётом локальных смещений)
	worldA := posA.Add(c.AnchorA)
	worldB := posB.Add(c.AnchorB)

	delta := worldB.Sub(worldA)
	dist := float32(math.Sqrt(float64(delta.X*delta.X + delta.Y*delta.Y + delta.Z*delta.Z)))
	if dist < 0.0001 {
		return
	}

	restLength := c.RestLength
	if restLength <= 0 {
		restLength = dist
	}

	diff := dist - restLength
	if diff == 0 {
		return
	}

	// Массы — kinematic = бесконечная масса
	invMassA := float32(0)
	if rbA != nil && !rbA.IsKinematic && rbA.Mass > 0 {
		invMassA = 1.0 / rbA.Mass
	}
	invMassB := float32(0)
	if rbB != nil && !rbB.IsKinematic && rbB.Mass > 0 {
		invMassB = 1.0 / rbB.Mass
	}

	totalInv := invMassA + invMassB
	if totalInv < 0.0001 {
		return
	}

	dir := emath.Vec3{
		X: delta.X / dist,
		Y: delta.Y / dist,
		Z: delta.Z / dist,
	}

	// Распределяем коррекцию по массам
	correction := dir.Mul(diff)
	corrA := correction.Mul(invMassA / totalInv)
	corrB := correction.Mul(-invMassB / totalInv)

	*posA = posA.Sub(corrA)
	*posB = posB.Add(corrB)

	// Velocity correction: устраняем расхождение скоростей вдоль оси ограничения
	if rbA != nil && rbB != nil && !rbA.IsKinematic && !rbB.IsKinematic {
		velAlongA := rbA.Velocity.X*dir.X + rbA.Velocity.Y*dir.Y + rbA.Velocity.Z*dir.Z
		velAlongB := rbB.Velocity.X*dir.X + rbB.Velocity.Y*dir.Y + rbB.Velocity.Z*dir.Z
		relVel := velAlongA - velAlongB
		if relVel != 0 {
			impulse := dir.Mul(-relVel / totalInv)
			rbA.Velocity = rbA.Velocity.Add(impulse.Mul(invMassA))
			rbB.Velocity = rbB.Velocity.Sub(impulse.Mul(invMassB))
		}
	}
}

// FixedPointConstraint — жёстко привязывает тело к мировой точке
type FixedPointConstraint struct {
	EntityID EntityID
	Point    emath.Vec3 // мировая точка
}

// Apply реализует Constraint
func (c *FixedPointConstraint) Apply(positions map[EntityID]*emath.Vec3, rigidbodies map[EntityID]*Rigidbody, iterations int) {
	pos := positions[c.EntityID]
	if pos == nil {
		return
	}
	*pos = c.Point
	rb := rigidbodies[c.EntityID]
	if rb != nil {
		rb.Velocity = emath.V3(0, 0, 0)
		rb.AngularVelocity = emath.V3(0, 0, 0)
	}
}

// ResolveConstraints применяет список ограничений несколько итераций
func ResolveConstraints(constraints []Constraint, positions map[EntityID]*emath.Vec3, rigidbodies map[EntityID]*Rigidbody, iterations int) {
	if iterations <= 0 {
		iterations = 3
	}
	for i := 0; i < iterations; i++ {
		for _, c := range constraints {
			c.Apply(positions, rigidbodies, iterations)
		}
	}
}
