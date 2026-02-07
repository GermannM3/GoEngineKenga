package render

import (
	"math"

	emath "goenginekenga/engine/math"
)

// Camera3D represents a 3D camera with projection
type Camera3D struct {
	Position emath.Vec3
	Target   emath.Vec3
	Up       emath.Vec3

	// Projection settings
	FOV         float32 // Field of view in degrees
	AspectRatio float32
	Near        float32
	Far         float32

	// Cached matrices
	viewMatrix       Matrix4
	projectionMatrix Matrix4
	viewProjMatrix   Matrix4
	dirty            bool
}

// Matrix4 represents a 4x4 matrix for 3D transformations
type Matrix4 [16]float32

// NewCamera3D creates a new 3D camera
func NewCamera3D() *Camera3D {
	return &Camera3D{
		Position:    emath.Vec3{X: 0, Y: 5, Z: 10},
		Target:      emath.Vec3{X: 0, Y: 0, Z: 0},
		Up:          emath.Vec3{X: 0, Y: 1, Z: 0},
		FOV:         60,
		AspectRatio: 16.0 / 9.0,
		Near:        0.1,
		Far:         1000,
		dirty:       true,
	}
}

// SetPosition sets the camera position
func (c *Camera3D) SetPosition(pos emath.Vec3) {
	c.Position = pos
	c.dirty = true
}

// SetTarget sets the camera look-at target
func (c *Camera3D) SetTarget(target emath.Vec3) {
	c.Target = target
	c.dirty = true
}

// SetFOV sets the field of view in degrees
func (c *Camera3D) SetFOV(fov float32) {
	c.FOV = fov
	c.dirty = true
}

// SetAspectRatio sets the aspect ratio
func (c *Camera3D) SetAspectRatio(aspect float32) {
	c.AspectRatio = aspect
	c.dirty = true
}

// GetViewMatrix returns the view matrix
func (c *Camera3D) GetViewMatrix() Matrix4 {
	c.updateMatrices()
	return c.viewMatrix
}

// GetProjectionMatrix returns the projection matrix
func (c *Camera3D) GetProjectionMatrix() Matrix4 {
	c.updateMatrices()
	return c.projectionMatrix
}

// GetViewProjectionMatrix returns combined view-projection matrix
func (c *Camera3D) GetViewProjectionMatrix() Matrix4 {
	c.updateMatrices()
	return c.viewProjMatrix
}

func (c *Camera3D) updateMatrices() {
	if !c.dirty {
		return
	}

	c.viewMatrix = LookAt(c.Position, c.Target, c.Up)
	c.projectionMatrix = Perspective(c.FOV, c.AspectRatio, c.Near, c.Far)
	c.viewProjMatrix = c.projectionMatrix.Multiply(c.viewMatrix)
	c.dirty = false
}

// Forward returns the camera's forward direction
func (c *Camera3D) Forward() emath.Vec3 {
	dir := emath.Vec3{
		X: c.Target.X - c.Position.X,
		Y: c.Target.Y - c.Position.Y,
		Z: c.Target.Z - c.Position.Z,
	}
	return Normalize3(dir)
}

// Right returns the camera's right direction
func (c *Camera3D) Right() emath.Vec3 {
	forward := c.Forward()
	return Normalize3(Cross(forward, c.Up))
}

// Identity4 returns an identity matrix
func Identity4() Matrix4 {
	return Matrix4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// LookAt creates a view matrix
func LookAt(eye, target, up emath.Vec3) Matrix4 {
	f := Normalize3(emath.Vec3{
		X: target.X - eye.X,
		Y: target.Y - eye.Y,
		Z: target.Z - eye.Z,
	})
	s := Normalize3(Cross(f, up))
	u := Cross(s, f)

	return Matrix4{
		s.X, u.X, -f.X, 0,
		s.Y, u.Y, -f.Y, 0,
		s.Z, u.Z, -f.Z, 0,
		-Dot(s, eye), -Dot(u, eye), Dot(f, eye), 1,
	}
}

// Perspective creates a perspective projection matrix
func Perspective(fovDegrees, aspect, near, far float32) Matrix4 {
	fovRad := fovDegrees * math.Pi / 180.0
	tanHalfFov := float32(math.Tan(float64(fovRad / 2)))

	return Matrix4{
		1 / (aspect * tanHalfFov), 0, 0, 0,
		0, 1 / tanHalfFov, 0, 0,
		0, 0, -(far + near) / (far - near), -1,
		0, 0, -(2 * far * near) / (far - near), 0,
	}
}

// Orthographic creates an orthographic projection matrix.
func Orthographic(left, right, bottom, top, near, far float32) Matrix4 {
	return Matrix4{
		2 / (right - left), 0, 0, 0,
		0, 2 / (top - bottom), 0, 0,
		0, 0, -2 / (far - near), 0,
		-(right + left) / (right - left), -(top + bottom) / (top - bottom), -(far + near) / (far - near), 1,
	}
}

// Translate creates a translation matrix
func Translate(v emath.Vec3) Matrix4 {
	return Matrix4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		v.X, v.Y, v.Z, 1,
	}
}

// Scale creates a scale matrix
func Scale(v emath.Vec3) Matrix4 {
	return Matrix4{
		v.X, 0, 0, 0,
		0, v.Y, 0, 0,
		0, 0, v.Z, 0,
		0, 0, 0, 1,
	}
}

// RotateY creates a rotation matrix around Y axis
func RotateY(angleDegrees float32) Matrix4 {
	rad := angleDegrees * math.Pi / 180.0
	c := float32(math.Cos(float64(rad)))
	s := float32(math.Sin(float64(rad)))

	return Matrix4{
		c, 0, s, 0,
		0, 1, 0, 0,
		-s, 0, c, 0,
		0, 0, 0, 1,
	}
}

// RotateX creates a rotation matrix around X axis
func RotateX(angleDegrees float32) Matrix4 {
	rad := angleDegrees * math.Pi / 180.0
	c := float32(math.Cos(float64(rad)))
	s := float32(math.Sin(float64(rad)))

	return Matrix4{
		1, 0, 0, 0,
		0, c, -s, 0,
		0, s, c, 0,
		0, 0, 0, 1,
	}
}

// RotateZ creates a rotation matrix around Z axis
func RotateZ(angleDegrees float32) Matrix4 {
	rad := angleDegrees * math.Pi / 180.0
	c := float32(math.Cos(float64(rad)))
	s := float32(math.Sin(float64(rad)))

	return Matrix4{
		c, -s, 0, 0,
		s, c, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// Multiply multiplies two matrices
func (m Matrix4) Multiply(other Matrix4) Matrix4 {
	var result Matrix4
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			sum := float32(0)
			for k := 0; k < 4; k++ {
				sum += m[i+k*4] * other[k+j*4]
			}
			result[i+j*4] = sum
		}
	}
	return result
}

// TransformPoint transforms a 3D point by this matrix
func (m Matrix4) TransformPoint(v emath.Vec3) emath.Vec3 {
	w := m[3]*v.X + m[7]*v.Y + m[11]*v.Z + m[15]
	if w == 0 {
		w = 1
	}
	return emath.Vec3{
		X: (m[0]*v.X + m[4]*v.Y + m[8]*v.Z + m[12]) / w,
		Y: (m[1]*v.X + m[5]*v.Y + m[9]*v.Z + m[13]) / w,
		Z: (m[2]*v.X + m[6]*v.Y + m[10]*v.Z + m[14]) / w,
	}
}

// TransformDirection transforms a direction vector (ignores translation)
func (m Matrix4) TransformDirection(v emath.Vec3) emath.Vec3 {
	return emath.Vec3{
		X: m[0]*v.X + m[4]*v.Y + m[8]*v.Z,
		Y: m[1]*v.X + m[5]*v.Y + m[9]*v.Z,
		Z: m[2]*v.X + m[6]*v.Y + m[10]*v.Z,
	}
}

// Helper functions
func Normalize3(v emath.Vec3) emath.Vec3 {
	len := float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
	if len < 0.0001 {
		return emath.Vec3{}
	}
	return emath.Vec3{X: v.X / len, Y: v.Y / len, Z: v.Z / len}
}

func Cross(a, b emath.Vec3) emath.Vec3 {
	return emath.Vec3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

func Dot(a, b emath.Vec3) float32 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

// Frustum planes from view-projection matrix (row-major).
// Plane equation: ax + by + cz + d = 0; normal points inward.
type Frustum struct {
	Planes [6][4]float32 // left, right, bottom, top, near, far
}

// ExtractFrustum builds frustum planes from view-projection matrix.
func ExtractFrustum(vp Matrix4) Frustum {
	var f Frustum
	m := vp
	// Left
	f.Planes[0][0] = m[3] + m[0]
	f.Planes[0][1] = m[7] + m[4]
	f.Planes[0][2] = m[11] + m[8]
	f.Planes[0][3] = m[15] + m[12]
	// Right
	f.Planes[1][0] = m[3] - m[0]
	f.Planes[1][1] = m[7] - m[4]
	f.Planes[1][2] = m[11] - m[8]
	f.Planes[1][3] = m[15] - m[12]
	// Bottom
	f.Planes[2][0] = m[3] + m[1]
	f.Planes[2][1] = m[7] + m[5]
	f.Planes[2][2] = m[11] + m[9]
	f.Planes[2][3] = m[15] + m[13]
	// Top
	f.Planes[3][0] = m[3] - m[1]
	f.Planes[3][1] = m[7] - m[5]
	f.Planes[3][2] = m[11] - m[9]
	f.Planes[3][3] = m[15] - m[13]
	// Near
	f.Planes[4][0] = m[3] + m[2]
	f.Planes[4][1] = m[7] + m[6]
	f.Planes[4][2] = m[11] + m[10]
	f.Planes[4][3] = m[15] + m[14]
	// Far
	f.Planes[5][0] = m[3] - m[2]
	f.Planes[5][1] = m[7] - m[6]
	f.Planes[5][2] = m[11] - m[10]
	f.Planes[5][3] = m[15] - m[14]
	// Normalize
	for i := 0; i < 6; i++ {
		len := float32(math.Sqrt(float64(f.Planes[i][0]*f.Planes[i][0] + f.Planes[i][1]*f.Planes[i][1] + f.Planes[i][2]*f.Planes[i][2])))
		if len > 0.0001 {
			f.Planes[i][0] /= len
			f.Planes[i][1] /= len
			f.Planes[i][2] /= len
			f.Planes[i][3] /= len
		}
	}
	return f
}

// SphereInFrustum returns true if sphere (center, radius) is inside or intersects frustum.
func (f *Frustum) SphereInFrustum(center emath.Vec3, radius float32) bool {
	for i := 0; i < 6; i++ {
		p := &f.Planes[i]
		d := p[0]*center.X + p[1]*center.Y + p[2]*center.Z + p[3]
		if d < -radius {
			return false
		}
	}
	return true
}
