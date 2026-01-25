package physics

import (
	"math"

	emath "goenginekenga/engine/math"
)

// EntityID is a type alias for entity identification (matches ecs.EntityID)
type EntityID = uint64

// CollisionInfo contains information about a collision
type CollisionInfo struct {
	EntityA      EntityID
	EntityB      EntityID
	Normal       emath.Vec3 // Collision normal (from A to B)
	Depth        float32    // Penetration depth
	ContactPoint emath.Vec3 // Point of contact
}

// AABB represents an axis-aligned bounding box
type AABB struct {
	Min emath.Vec3
	Max emath.Vec3
}

// NewAABB creates an AABB from center and half-extents
func NewAABB(center, halfExtents emath.Vec3) AABB {
	return AABB{
		Min: center.Sub(halfExtents),
		Max: center.Add(halfExtents),
	}
}

// AABBFromCollider creates an AABB from a collider and position
func AABBFromCollider(c *Collider, pos emath.Vec3) AABB {
	worldCenter := pos.Add(c.Center)
	switch c.Type {
	case "box":
		halfSize := emath.Vec3{X: c.Size.X / 2, Y: c.Size.Y / 2, Z: c.Size.Z / 2}
		return NewAABB(worldCenter, halfSize)
	case "sphere":
		r := emath.Vec3{X: c.Radius, Y: c.Radius, Z: c.Radius}
		return NewAABB(worldCenter, r)
	case "capsule":
		// Approximate as AABB
		r := c.Radius
		h := c.Height / 2
		halfSize := emath.Vec3{X: r, Y: h + r, Z: r}
		return NewAABB(worldCenter, halfSize)
	default:
		return NewAABB(worldCenter, emath.Vec3{X: 0.5, Y: 0.5, Z: 0.5})
	}
}

// Intersects checks if two AABBs intersect
func (a AABB) Intersects(b AABB) bool {
	return a.Min.X <= b.Max.X && a.Max.X >= b.Min.X &&
		a.Min.Y <= b.Max.Y && a.Max.Y >= b.Min.Y &&
		a.Min.Z <= b.Max.Z && a.Max.Z >= b.Min.Z
}

// CheckAABB checks collision between two box colliders
func CheckAABB(a, b *Collider, posA, posB emath.Vec3, idA, idB EntityID) *CollisionInfo {
	// Calculate world centers
	centerA := posA.Add(a.Center)
	centerB := posB.Add(b.Center)

	// Half sizes
	halfA := emath.Vec3{X: a.Size.X / 2, Y: a.Size.Y / 2, Z: a.Size.Z / 2}
	halfB := emath.Vec3{X: b.Size.X / 2, Y: b.Size.Y / 2, Z: b.Size.Z / 2}

	// Calculate overlap on each axis
	diff := centerB.Sub(centerA)

	overlapX := (halfA.X + halfB.X) - abs(diff.X)
	overlapY := (halfA.Y + halfB.Y) - abs(diff.Y)
	overlapZ := (halfA.Z + halfB.Z) - abs(diff.Z)

	// No collision if any axis has no overlap
	if overlapX <= 0 || overlapY <= 0 || overlapZ <= 0 {
		return nil
	}

	// Find minimum overlap axis (this is the collision normal)
	var normal emath.Vec3
	var depth float32

	if overlapX <= overlapY && overlapX <= overlapZ {
		depth = overlapX
		if diff.X > 0 {
			normal = emath.Vec3{X: 1, Y: 0, Z: 0}
		} else {
			normal = emath.Vec3{X: -1, Y: 0, Z: 0}
		}
	} else if overlapY <= overlapX && overlapY <= overlapZ {
		depth = overlapY
		if diff.Y > 0 {
			normal = emath.Vec3{X: 0, Y: 1, Z: 0}
		} else {
			normal = emath.Vec3{X: 0, Y: -1, Z: 0}
		}
	} else {
		depth = overlapZ
		if diff.Z > 0 {
			normal = emath.Vec3{X: 0, Y: 0, Z: 1}
		} else {
			normal = emath.Vec3{X: 0, Y: 0, Z: -1}
		}
	}

	// Calculate contact point (midpoint of overlap region)
	contactPoint := centerA.Add(normal.Mul(halfA.X)) // Simplified

	return &CollisionInfo{
		EntityA:      idA,
		EntityB:      idB,
		Normal:       normal,
		Depth:        depth,
		ContactPoint: contactPoint,
	}
}

// CheckSphereSphere checks collision between two sphere colliders
func CheckSphereSphere(a, b *Collider, posA, posB emath.Vec3, idA, idB EntityID) *CollisionInfo {
	centerA := posA.Add(a.Center)
	centerB := posB.Add(b.Center)

	diff := centerB.Sub(centerA)
	distSq := diff.X*diff.X + diff.Y*diff.Y + diff.Z*diff.Z
	radiusSum := a.Radius + b.Radius

	if distSq >= radiusSum*radiusSum {
		return nil // No collision
	}

	dist := float32(math.Sqrt(float64(distSq)))
	if dist < 0.0001 {
		// Centers are at same position
		return &CollisionInfo{
			EntityA:      idA,
			EntityB:      idB,
			Normal:       emath.Vec3{X: 0, Y: 1, Z: 0},
			Depth:        radiusSum,
			ContactPoint: centerA,
		}
	}

	normal := emath.Vec3{
		X: diff.X / dist,
		Y: diff.Y / dist,
		Z: diff.Z / dist,
	}

	depth := radiusSum - dist
	contactPoint := centerA.Add(normal.Mul(a.Radius))

	return &CollisionInfo{
		EntityA:      idA,
		EntityB:      idB,
		Normal:       normal,
		Depth:        depth,
		ContactPoint: contactPoint,
	}
}

// CheckBoxSphere checks collision between a box and a sphere
func CheckBoxSphere(box, sphere *Collider, boxPos, spherePos emath.Vec3, idBox, idSphere EntityID) *CollisionInfo {
	boxCenter := boxPos.Add(box.Center)
	sphereCenter := spherePos.Add(sphere.Center)

	halfBox := emath.Vec3{X: box.Size.X / 2, Y: box.Size.Y / 2, Z: box.Size.Z / 2}

	// Find closest point on box to sphere center
	closest := emath.Vec3{
		X: clamp(sphereCenter.X, boxCenter.X-halfBox.X, boxCenter.X+halfBox.X),
		Y: clamp(sphereCenter.Y, boxCenter.Y-halfBox.Y, boxCenter.Y+halfBox.Y),
		Z: clamp(sphereCenter.Z, boxCenter.Z-halfBox.Z, boxCenter.Z+halfBox.Z),
	}

	diff := sphereCenter.Sub(closest)
	distSq := diff.X*diff.X + diff.Y*diff.Y + diff.Z*diff.Z

	if distSq >= sphere.Radius*sphere.Radius {
		return nil // No collision
	}

	dist := float32(math.Sqrt(float64(distSq)))
	if dist < 0.0001 {
		// Sphere center is inside box
		return &CollisionInfo{
			EntityA:      idBox,
			EntityB:      idSphere,
			Normal:       emath.Vec3{X: 0, Y: 1, Z: 0},
			Depth:        sphere.Radius,
			ContactPoint: closest,
		}
	}

	normal := emath.Vec3{
		X: diff.X / dist,
		Y: diff.Y / dist,
		Z: diff.Z / dist,
	}

	return &CollisionInfo{
		EntityA:      idBox,
		EntityB:      idSphere,
		Normal:       normal,
		Depth:        sphere.Radius - dist,
		ContactPoint: closest,
	}
}

// CheckCollision checks collision between two colliders of any type
func CheckCollision(a, b *Collider, posA, posB emath.Vec3, idA, idB EntityID) *CollisionInfo {
	// First, broad phase check with AABBs
	aabbA := AABBFromCollider(a, posA)
	aabbB := AABBFromCollider(b, posB)

	if !aabbA.Intersects(aabbB) {
		return nil
	}

	// Narrow phase - specific collision checks
	typeA := a.Type
	typeB := b.Type

	switch {
	case typeA == "box" && typeB == "box":
		return CheckAABB(a, b, posA, posB, idA, idB)
	case typeA == "sphere" && typeB == "sphere":
		return CheckSphereSphere(a, b, posA, posB, idA, idB)
	case typeA == "box" && typeB == "sphere":
		return CheckBoxSphere(a, b, posA, posB, idA, idB)
	case typeA == "sphere" && typeB == "box":
		info := CheckBoxSphere(b, a, posB, posA, idB, idA)
		if info != nil {
			// Swap entities and invert normal
			info.EntityA, info.EntityB = idA, idB
			info.Normal = info.Normal.Mul(-1)
		}
		return info
	default:
		// Fallback to AABB for unsupported types
		if aabbA.Intersects(aabbB) {
			return &CollisionInfo{
				EntityA:      idA,
				EntityB:      idB,
				Normal:       emath.Vec3{X: 0, Y: 1, Z: 0},
				Depth:        0.1,
				ContactPoint: posA.Add(posB).Mul(0.5),
			}
		}
		return nil
	}
}

// ResolveCollision applies collision response to two bodies
func ResolveCollision(info *CollisionInfo, rbA, rbB *Rigidbody, posA, posB *emath.Vec3) {
	if info == nil {
		return
	}

	// Determine which bodies are dynamic
	aIsStatic := rbA == nil || rbA.IsKinematic || rbA.Mass <= 0
	bIsStatic := rbB == nil || rbB.IsKinematic || rbB.Mass <= 0

	if aIsStatic && bIsStatic {
		return // Both static, no response needed
	}

	// Calculate separation
	separationA := float32(0)
	separationB := float32(0)

	if aIsStatic {
		separationB = info.Depth
	} else if bIsStatic {
		separationA = info.Depth
	} else {
		// Distribute based on mass
		totalMass := rbA.Mass + rbB.Mass
		separationA = info.Depth * (rbB.Mass / totalMass)
		separationB = info.Depth * (rbA.Mass / totalMass)
	}

	// Apply position correction
	if !aIsStatic && posA != nil {
		*posA = posA.Sub(info.Normal.Mul(separationA))
	}
	if !bIsStatic && posB != nil {
		*posB = posB.Add(info.Normal.Mul(separationB))
	}

	// Apply impulse for velocity response
	if rbA != nil && rbB != nil && !aIsStatic && !bIsStatic {
		// Relative velocity
		relVel := rbB.Velocity.Sub(rbA.Velocity)

		// Relative velocity along normal
		velAlongNormal := relVel.X*info.Normal.X + relVel.Y*info.Normal.Y + relVel.Z*info.Normal.Z

		// Don't resolve if velocities are separating
		if velAlongNormal > 0 {
			return
		}

		// Restitution (bounciness) - use average
		restitution := float32(0.3) // Default bounce

		// Calculate impulse scalar
		invMassA := float32(1.0) / rbA.Mass
		invMassB := float32(1.0) / rbB.Mass
		j := -(1 + restitution) * velAlongNormal / (invMassA + invMassB)

		// Apply impulse
		impulse := info.Normal.Mul(j)
		rbA.Velocity = rbA.Velocity.Sub(impulse.Mul(invMassA))
		rbB.Velocity = rbB.Velocity.Add(impulse.Mul(invMassB))
	} else if rbA != nil && !aIsStatic && bIsStatic {
		// A bounces off static B
		velAlongNormal := rbA.Velocity.X*info.Normal.X + rbA.Velocity.Y*info.Normal.Y + rbA.Velocity.Z*info.Normal.Z
		if velAlongNormal < 0 {
			restitution := float32(0.3)
			rbA.Velocity = rbA.Velocity.Sub(info.Normal.Mul((1 + restitution) * velAlongNormal))
		}
	} else if rbB != nil && !bIsStatic && aIsStatic {
		// B bounces off static A
		velAlongNormal := rbB.Velocity.X*info.Normal.X + rbB.Velocity.Y*info.Normal.Y + rbB.Velocity.Z*info.Normal.Z
		if velAlongNormal > 0 {
			restitution := float32(0.3)
			rbB.Velocity = rbB.Velocity.Add(info.Normal.Mul((1 + restitution) * velAlongNormal))
		}
	}
}

// Helper functions
func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func clamp(v, minVal, maxVal float32) float32 {
	if v < minVal {
		return minVal
	}
	if v > maxVal {
		return maxVal
	}
	return v
}

// SpatialHash provides broad-phase collision detection
type SpatialHash struct {
	cellSize float32
	cells    map[int64][]EntityID
}

// NewSpatialHash creates a new spatial hash
func NewSpatialHash(cellSize float32) *SpatialHash {
	return &SpatialHash{
		cellSize: cellSize,
		cells:    make(map[int64][]EntityID),
	}
}

// Clear removes all entries
func (h *SpatialHash) Clear() {
	h.cells = make(map[int64][]EntityID)
}

// hashKey generates a hash key for a cell position
func (h *SpatialHash) hashKey(x, y, z int) int64 {
	// Simple hash combining x, y, z
	return int64(x)*73856093 ^ int64(y)*19349663 ^ int64(z)*83492791
}

// Insert adds an entity to the spatial hash
func (h *SpatialHash) Insert(id EntityID, aabb AABB) {
	minX := int(aabb.Min.X / h.cellSize)
	minY := int(aabb.Min.Y / h.cellSize)
	minZ := int(aabb.Min.Z / h.cellSize)
	maxX := int(aabb.Max.X / h.cellSize)
	maxY := int(aabb.Max.Y / h.cellSize)
	maxZ := int(aabb.Max.Z / h.cellSize)

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				key := h.hashKey(x, y, z)
				h.cells[key] = append(h.cells[key], id)
			}
		}
	}
}

// Query returns all entities that might collide with the given AABB
func (h *SpatialHash) Query(aabb AABB) []EntityID {
	var result []EntityID
	seen := make(map[EntityID]bool)

	minX := int(aabb.Min.X / h.cellSize)
	minY := int(aabb.Min.Y / h.cellSize)
	minZ := int(aabb.Min.Z / h.cellSize)
	maxX := int(aabb.Max.X / h.cellSize)
	maxY := int(aabb.Max.Y / h.cellSize)
	maxZ := int(aabb.Max.Z / h.cellSize)

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				key := h.hashKey(x, y, z)
				for _, id := range h.cells[key] {
					if !seen[id] {
						seen[id] = true
						result = append(result, id)
					}
				}
			}
		}
	}

	return result
}

// CollisionManager handles all collision detection and response
type CollisionManager struct {
	spatialHash *SpatialHash

	// Callbacks
	OnCollisionEnter func(a, b EntityID, info *CollisionInfo)
	OnCollisionExit  func(a, b EntityID)
	OnTriggerEnter   func(a, b EntityID)
	OnTriggerExit    func(a, b EntityID)

	// Track active collisions for enter/exit events
	activeCollisions map[collisionPair]bool
	prevCollisions   map[collisionPair]bool
}

type collisionPair struct {
	a, b EntityID
}

func makeCollisionPair(a, b EntityID) collisionPair {
	if a > b {
		return collisionPair{b, a}
	}
	return collisionPair{a, b}
}

// NewCollisionManager creates a new collision manager
func NewCollisionManager(cellSize float32) *CollisionManager {
	return &CollisionManager{
		spatialHash:      NewSpatialHash(cellSize),
		activeCollisions: make(map[collisionPair]bool),
		prevCollisions:   make(map[collisionPair]bool),
	}
}

// ColliderData holds collider information for collision detection
type ColliderData struct {
	ID       EntityID
	Collider *Collider
	Position emath.Vec3
}

// DetectAndResolve performs collision detection and resolution
func (cm *CollisionManager) DetectAndResolve(
	colliders []ColliderData,
	positions map[EntityID]*emath.Vec3,
	rigidbodies map[EntityID]*Rigidbody,
) []CollisionInfo {
	// Swap collision maps
	cm.prevCollisions = cm.activeCollisions
	cm.activeCollisions = make(map[collisionPair]bool)

	var collisions []CollisionInfo

	// Build spatial hash
	cm.spatialHash.Clear()
	for _, e := range colliders {
		pos := e.Position
		if p, ok := positions[e.ID]; ok && p != nil {
			pos = *p
		}
		aabb := AABBFromCollider(e.Collider, pos)
		cm.spatialHash.Insert(e.ID, aabb)
	}

	// Create lookup map
	colliderMap := make(map[EntityID]*ColliderData)
	for i := range colliders {
		colliderMap[colliders[i].ID] = &colliders[i]
	}

	// Check collisions
	checked := make(map[collisionPair]bool)

	for _, entityA := range colliders {
		posA := entityA.Position
		if p, ok := positions[entityA.ID]; ok && p != nil {
			posA = *p
		}

		aabb := AABBFromCollider(entityA.Collider, posA)
		candidates := cm.spatialHash.Query(aabb)

		for _, idB := range candidates {
			if entityA.ID == idB {
				continue
			}

			pair := makeCollisionPair(entityA.ID, idB)
			if checked[pair] {
				continue
			}
			checked[pair] = true

			// Find entity B data
			entityB, ok := colliderMap[idB]
			if !ok || entityB == nil {
				continue
			}

			posB := entityB.Position
			if p, ok := positions[idB]; ok && p != nil {
				posB = *p
			}

			// Check collision
			info := CheckCollision(entityA.Collider, entityB.Collider, posA, posB, entityA.ID, idB)
			if info != nil {
				collisions = append(collisions, *info)
				cm.activeCollisions[pair] = true

				// Check for triggers
				if entityA.Collider.IsTrigger || entityB.Collider.IsTrigger {
					if !cm.prevCollisions[pair] && cm.OnTriggerEnter != nil {
						cm.OnTriggerEnter(entityA.ID, idB)
					}
				} else {
					// Resolve physical collision
					rbA := rigidbodies[entityA.ID]
					rbB := rigidbodies[idB]
					pA := positions[entityA.ID]
					pB := positions[idB]
					ResolveCollision(info, rbA, rbB, pA, pB)

					if !cm.prevCollisions[pair] && cm.OnCollisionEnter != nil {
						cm.OnCollisionEnter(entityA.ID, idB, info)
					}
				}
			}
		}
	}

	// Fire exit events
	for pair := range cm.prevCollisions {
		if !cm.activeCollisions[pair] {
			if cm.OnCollisionExit != nil {
				cm.OnCollisionExit(pair.a, pair.b)
			}
			if cm.OnTriggerExit != nil {
				cm.OnTriggerExit(pair.a, pair.b)
			}
		}
	}

	return collisions
}
