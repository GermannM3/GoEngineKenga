package render

import (
	"math"

	emath "goenginekenga/engine/math"
)

// OrbitState хранит состояние orbit-камеры (вращение вокруг точки, зум, панорама)
type OrbitState struct {
	Target   emath.Vec3 // точка, вокруг которой вращаемся
	Distance float32    // расстояние от камеры до target
	Yaw      float32    // градусы, вращение вокруг Y (горизонталь)
	Pitch    float32    // градусы, вращение вокруг X (вертикаль, -89..89)

	// Настройки
	OrbitSpeed  float32 // чувствительность вращения (градусов/пиксель)
	PanSpeed    float32 // чувствительность панорамы
	ZoomSpeed   float32 // множитель зума от scroll
	MinDistance float32
	MaxDistance float32
}

// DefaultOrbitState возвращает разумные настройки по умолчанию
func DefaultOrbitState() OrbitState {
	return OrbitState{
		Target:      emath.V3(0, 0, 0),
		Distance:    10,
		Yaw:         0,
		Pitch:       20,
		OrbitSpeed:  0.3,
		PanSpeed:    0.01,
		ZoomSpeed:   1.1,
		MinDistance: 1,
		MaxDistance: 200,
	}
}

// Position возвращает позицию камеры по текущему состоянию orbit
func (o *OrbitState) Position() emath.Vec3 {
	dir := o.forward()
	return emath.Vec3{
		X: o.Target.X + dir.X*o.Distance,
		Y: o.Target.Y + dir.Y*o.Distance,
		Z: o.Target.Z + dir.Z*o.Distance,
	}
}

// forward возвращает единичный вектор направления взгляда (от камеры к target)
func (o *OrbitState) forward() emath.Vec3 {
	radY := float64(o.Yaw) * math.Pi / 180
	radP := float64(o.Pitch) * math.Pi / 180
	cosP := float32(math.Cos(radP))
	return emath.Vec3{
		X: float32(math.Sin(radY)) * cosP,
		Y: float32(-math.Sin(radP)),
		Z: float32(-math.Cos(radY)) * cosP,
	}
}

// Orbit применяет вращение (deltaX, deltaY в пикселях)
func (o *OrbitState) Orbit(deltaX, deltaY float32) {
	o.Yaw -= deltaX * o.OrbitSpeed
	o.Pitch += deltaY * o.OrbitSpeed
	if o.Pitch > 89 {
		o.Pitch = 89
	}
	if o.Pitch < -89 {
		o.Pitch = -89
	}
}

// Pan применяет панораму (сдвиг target в плоскости экрана)
func (o *OrbitState) Pan(deltaX, deltaY float32) {
	dir := o.forward()
	worldUp := emath.Vec3{X: 0, Y: 1, Z: 0}
	right := emath.Vec3{X: dir.Z, Y: 0, Z: -dir.X}
	if n := float32(math.Sqrt(float64(right.X*right.X + right.Z*right.Z))); n > 0.0001 {
		right.X /= n
		right.Z /= n
	}
	up := emath.Vec3{
		X: right.Z*dir.Y - right.Y*dir.Z,
		Y: right.X*dir.Z - right.Z*dir.X,
		Z: right.Y*dir.X - right.X*dir.Y,
	}
	if n := float32(math.Sqrt(float64(up.X*up.X + up.Y*up.Y + up.Z*up.Z))); n > 0.0001 {
		up.X /= n
		up.Y /= n
		up.Z /= n
	} else {
		up = worldUp
	}
	scale := o.Distance * o.PanSpeed
	o.Target.X += right.X*deltaX*scale - up.X*deltaY*scale
	o.Target.Y += right.Y*deltaX*scale - up.Y*deltaY*scale
	o.Target.Z += right.Z*deltaX*scale - up.Z*deltaY*scale
}

// Zoom применяет зум (положительный = приближение)
func (o *OrbitState) Zoom(delta float32) {
	if delta > 0 {
		o.Distance /= o.ZoomSpeed
	} else {
		o.Distance *= o.ZoomSpeed
	}
	if o.Distance < o.MinDistance {
		o.Distance = o.MinDistance
	}
	if o.Distance > o.MaxDistance {
		o.Distance = o.MaxDistance
	}
}

// SyncFromTransform восстанавливает orbit state из позиции и rotation камеры
func (o *OrbitState) SyncFromTransform(pos emath.Vec3, rotY, rotX float32) {
	o.Yaw = rotY
	o.Pitch = rotX
	radY := float64(rotY) * math.Pi / 180
	radP := float64(rotX) * math.Pi / 180
	forward := emath.Vec3{
		X: float32(math.Sin(radY) * math.Cos(radP)),
		Y: float32(-math.Sin(radP)),
		Z: float32(-math.Cos(radY) * math.Cos(radP)),
	}
	// Расстояние примем 10 если не можем вычислить
	o.Distance = 10
	o.Target = emath.Vec3{
		X: pos.X - forward.X*o.Distance,
		Y: pos.Y - forward.Y*o.Distance,
		Z: pos.Z - forward.Z*o.Distance,
	}
}
