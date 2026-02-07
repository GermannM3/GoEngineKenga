package physics

import (
	"math"

	emath "goenginekenga/engine/math"
)

// RaycastHit содержит результат raycast
type RaycastHit struct {
	EntityID     EntityID
	Point        emath.Vec3
	Normal       emath.Vec3
	Distance     float32
	Collider     *Collider
	ColliderPos  emath.Vec3
}

// Raycast проверяет пересечение луча с коллайдерами. Возвращает ближайшее попадание или nil.
func Raycast(origin, direction emath.Vec3, maxDist float32, colliders []ColliderData, positions map[EntityID]emath.Vec3) *RaycastHit {
	dir := direction
	lenDir := dir.Len()
	if lenDir < 0.0001 {
		return nil
	}
	dir = dir.Mul(1.0 / lenDir)

	var nearest *RaycastHit
	nearestDist := maxDist

	for _, c := range colliders {
		pos := c.Position
		if p, ok := positions[c.ID]; ok {
			pos = p
		}
		dist, point, normal := rayCollider(origin, dir, &c, pos)
		if dist >= 0 && dist < nearestDist {
			nearestDist = dist
			nearest = &RaycastHit{
				EntityID:    c.ID,
				Point:       point,
				Normal:      normal,
				Distance:    dist,
				Collider:    c.Collider,
				ColliderPos: pos,
			}
		}
	}

	return nearest
}

// rayCollider возвращает distance (>=0 если hit), point, normal. -1 если miss.
func rayCollider(origin, dir emath.Vec3, c *ColliderData, pos emath.Vec3) (float32, emath.Vec3, emath.Vec3) {
	center := pos.Add(c.Collider.Center)
	switch c.Collider.Type {
	case "sphere":
		return raySphere(origin, dir, center, c.Collider.Radius)
	case "box":
		return rayBox(origin, dir, center, c.Collider.Size)
	case "capsule":
		return rayCapsule(origin, dir, center, c.Collider.Radius, c.Collider.Height)
	default:
		return raySphere(origin, dir, center, 0.5)
	}
}

func raySphere(origin, dir, center emath.Vec3, radius float32) (float32, emath.Vec3, emath.Vec3) {
	oc := origin.Sub(center)
	a := dir.X*dir.X + dir.Y*dir.Y + dir.Z*dir.Z
	b := 2 * (oc.X*dir.X + oc.Y*dir.Y + oc.Z*dir.Z)
	c := oc.X*oc.X + oc.Y*oc.Y + oc.Z*oc.Z - radius*radius
	disc := b*b - 4*a*c
	if disc < 0 {
		return -1, emath.Vec3{}, emath.Vec3{}
	}
	sqrt := float32(math.Sqrt(float64(disc)))
	t := (-b - sqrt) / (2 * a)
	if t < 0 {
		t = (-b + sqrt) / (2 * a)
	}
	if t < 0 {
		return -1, emath.Vec3{}, emath.Vec3{}
	}
	point := origin.Add(dir.Mul(t))
	normal := point.Sub(center).Mul(1.0 / radius)
	return t, point, normal
}

func rayBox(origin, dir, center emath.Vec3, size emath.Vec3) (float32, emath.Vec3, emath.Vec3) {
	half := emath.Vec3{X: size.X / 2, Y: size.Y / 2, Z: size.Z / 2}
	min := center.Sub(half)
	max := center.Add(half)
	eps := float32(1e-7)

	inf := float32(1e9)
	invX := inf
	if abs32(dir.X) > eps {
		invX = 1 / dir.X
	}
	invY := inf
	if abs32(dir.Y) > eps {
		invY = 1 / dir.Y
	}
	invZ := inf
	if abs32(dir.Z) > eps {
		invZ = 1 / dir.Z
	}

	var t1, t2 float32
	if abs32(dir.X) < eps {
		if origin.X < min.X || origin.X > max.X {
			return -1, emath.Vec3{}, emath.Vec3{}
		}
		t1, t2 = -inf, inf
	} else {
		t1 = (min.X - origin.X) * invX
		t2 = (max.X - origin.X) * invX
		if t1 > t2 {
			t1, t2 = t2, t1
		}
	}
	var t3, t4, t5, t6 float32
	if abs32(dir.Y) < eps {
		if origin.Y < min.Y || origin.Y > max.Y {
			return -1, emath.Vec3{}, emath.Vec3{}
		}
		t3, t4 = -inf, inf
	} else {
		t3 = (min.Y - origin.Y) * invY
		t4 = (max.Y - origin.Y) * invY
		if t3 > t4 {
			t3, t4 = t4, t3
		}
	}
	if abs32(dir.Z) < eps {
		if origin.Z < min.Z || origin.Z > max.Z {
			return -1, emath.Vec3{}, emath.Vec3{}
		}
		t5, t6 = -inf, inf
	} else {
		t5 = (min.Z - origin.Z) * invZ
		t6 = (max.Z - origin.Z) * invZ
		if t5 > t6 {
			t5, t6 = t6, t5
		}
	}

	tmin := max3(min2(t1, t2), min2(t3, t4), min2(t5, t6))
	tmax := min3(max2(t1, t2), max2(t3, t4), max2(t5, t6))

	if tmin > tmax || tmax < 0 {
		return -1, emath.Vec3{}, emath.Vec3{}
	}
	t := tmin
	if t < 0 {
		t = tmax
	}
	if t < 0 {
		return -1, emath.Vec3{}, emath.Vec3{}
	}
	point := origin.Add(dir.Mul(t))

	var normal emath.Vec3
	u := (point.X - min.X) / size.X
	if u < 0.01 {
		normal.X = -1
	} else if u > 0.99 {
		normal.X = 1
	}
	v := (point.Y - min.Y) / size.Y
	if v < 0.01 {
		normal.Y = -1
	} else if v > 0.99 {
		normal.Y = 1
	}
	w := (point.Z - min.Z) / size.Z
	if w < 0.01 {
		normal.Z = -1
	} else if w > 0.99 {
		normal.Z = 1
	}
	if normal.X == 0 && normal.Y == 0 && normal.Z == 0 {
		normal.Y = 1
	}
	return t, point, normal
}

func rayCapsule(origin, dir, center emath.Vec3, radius, height float32) (float32, emath.Vec3, emath.Vec3) {
	halfH := height / 2
	p1 := center.Add(emath.Vec3{X: 0, Y: -halfH, Z: 0})
	p2 := center.Add(emath.Vec3{X: 0, Y: halfH, Z: 0})
	eps := float32(0.0001)

	// Binary search for first t where dist(origin+t*dir, segment) <= radius
	distAt := func(t float32) float32 {
		pt := origin.Add(dir.Mul(t))
		cp := closestPointOnSegment(pt, p1, p2)
		return pt.Sub(cp).Len()
	}

	// Find t of closest approach
	tClosest := float32(0)
	bestDist := float32(1e9)
	for t := float32(0); t < 1000; t += 0.5 {
		d := distAt(t)
		if d < bestDist {
			bestDist = d
			tClosest = t
		}
		if d > bestDist && t > tClosest+0.5 {
			break
		}
	}

	if bestDist > radius {
		return -1, emath.Vec3{}, emath.Vec3{}
	}

	// Binary search for entry point (smallest t where dist <= radius)
	tLo, tHi := float32(0), tClosest
	if distAt(0) <= radius {
		tLo = 0
	} else {
		for tHi-tLo > 0.001 {
			tMid := (tLo + tHi) / 2
			if distAt(tMid) <= radius {
				tHi = tMid
			} else {
				tLo = tMid
			}
		}
	}
	tHit := tHi
	point := origin.Add(dir.Mul(tHit))
	cp := closestPointOnSegment(point, p1, p2)
	normal := point.Sub(cp)
	if normal.Len() > eps {
		normal = normal.Mul(1.0 / normal.Len())
	} else {
		normal = emath.Vec3{X: 0, Y: 1, Z: 0}
	}
	return tHit, point, normal
}

func min2(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
func max2(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
func max3(a, b, c float32) float32 {
	if a > b && a > c {
		return a
	}
	if b > c {
		return b
	}
	return c
}
func min3(a, b, c float32) float32 {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}
func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func clamp3(lo, hi, v float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
