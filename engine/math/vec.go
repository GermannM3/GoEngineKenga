package math

import "math"

type Vec2 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type Vec3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

func V2(x, y float32) Vec2    { return Vec2{X: x, Y: y} }
func V3(x, y, z float32) Vec3 { return Vec3{X: x, Y: y, Z: z} }

func (v Vec3) Add(o Vec3) Vec3    { return Vec3{X: v.X + o.X, Y: v.Y + o.Y, Z: v.Z + o.Z} }
func (v Vec3) Sub(o Vec3) Vec3    { return Vec3{X: v.X - o.X, Y: v.Y - o.Y, Z: v.Z - o.Z} }
func (v Vec3) Mul(s float32) Vec3 { return Vec3{X: v.X * s, Y: v.Y * s, Z: v.Z * s} }

func (v Vec3) Len() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
}
