package render

import (
	"math"

	emath "goenginekenga/engine/math"
)

// Mesh represents a 3D mesh
type Mesh struct {
	Vertices []float32 // x, y, z
	Normals  []float32 // nx, ny, nz
	UVs      []float32 // u, v
	Indices  []uint32
	Colors   []float32 // r, g, b, a (per vertex)
}

// NewMesh creates an empty mesh
func NewMesh() *Mesh {
	return &Mesh{}
}

// VertexCount returns the number of vertices
func (m *Mesh) VertexCount() int {
	return len(m.Vertices) / 3
}

// Clone creates a copy of the mesh
func (m *Mesh) Clone() *Mesh {
	clone := &Mesh{
		Vertices: make([]float32, len(m.Vertices)),
		Normals:  make([]float32, len(m.Normals)),
		UVs:      make([]float32, len(m.UVs)),
		Indices:  make([]uint32, len(m.Indices)),
		Colors:   make([]float32, len(m.Colors)),
	}
	copy(clone.Vertices, m.Vertices)
	copy(clone.Normals, m.Normals)
	copy(clone.UVs, m.UVs)
	copy(clone.Indices, m.Indices)
	copy(clone.Colors, m.Colors)
	return clone
}

// Primitive mesh generators

// CreateCube creates a unit cube mesh
func CreateCube() *Mesh {
	m := &Mesh{
		Vertices: []float32{
			// Front face
			-0.5, -0.5, 0.5, 0.5, -0.5, 0.5, 0.5, 0.5, 0.5, -0.5, 0.5, 0.5,
			// Back face
			0.5, -0.5, -0.5, -0.5, -0.5, -0.5, -0.5, 0.5, -0.5, 0.5, 0.5, -0.5,
			// Top face
			-0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, -0.5, -0.5, 0.5, -0.5,
			// Bottom face
			-0.5, -0.5, -0.5, 0.5, -0.5, -0.5, 0.5, -0.5, 0.5, -0.5, -0.5, 0.5,
			// Right face
			0.5, -0.5, 0.5, 0.5, -0.5, -0.5, 0.5, 0.5, -0.5, 0.5, 0.5, 0.5,
			// Left face
			-0.5, -0.5, -0.5, -0.5, -0.5, 0.5, -0.5, 0.5, 0.5, -0.5, 0.5, -0.5,
		},
		Normals: []float32{
			// Front
			0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1,
			// Back
			0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1,
			// Top
			0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0,
			// Bottom
			0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0,
			// Right
			1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0,
			// Left
			-1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0,
		},
		UVs: []float32{
			0, 0, 1, 0, 1, 1, 0, 1,
			0, 0, 1, 0, 1, 1, 0, 1,
			0, 0, 1, 0, 1, 1, 0, 1,
			0, 0, 1, 0, 1, 1, 0, 1,
			0, 0, 1, 0, 1, 1, 0, 1,
			0, 0, 1, 0, 1, 1, 0, 1,
		},
		Indices: []uint32{
			0, 1, 2, 0, 2, 3, // Front
			4, 5, 6, 4, 6, 7, // Back
			8, 9, 10, 8, 10, 11, // Top
			12, 13, 14, 12, 14, 15, // Bottom
			16, 17, 18, 16, 18, 19, // Right
			20, 21, 22, 20, 22, 23, // Left
		},
	}
	return m
}

// CreateSphere creates a UV sphere
func CreateSphere(segments, rings int) *Mesh {
	m := &Mesh{}

	for y := 0; y <= rings; y++ {
		for x := 0; x <= segments; x++ {
			xSegment := float64(x) / float64(segments)
			ySegment := float64(y) / float64(rings)

			px := float32(math.Cos(xSegment*2*math.Pi) * math.Sin(ySegment*math.Pi))
			py := float32(math.Cos(ySegment * math.Pi))
			pz := float32(math.Sin(xSegment*2*math.Pi) * math.Sin(ySegment*math.Pi))

			m.Vertices = append(m.Vertices, px*0.5, py*0.5, pz*0.5)
			m.Normals = append(m.Normals, px, py, pz)
			m.UVs = append(m.UVs, float32(xSegment), float32(ySegment))
		}
	}

	for y := 0; y < rings; y++ {
		for x := 0; x < segments; x++ {
			i0 := uint32(y*(segments+1) + x)
			i1 := i0 + uint32(segments) + 1

			m.Indices = append(m.Indices, i0, i1, i0+1)
			m.Indices = append(m.Indices, i1, i1+1, i0+1)
		}
	}

	return m
}

// CreatePlane creates a plane mesh
func CreatePlane(width, height float32, segmentsX, segmentsZ int) *Mesh {
	m := &Mesh{}

	for z := 0; z <= segmentsZ; z++ {
		for x := 0; x <= segmentsX; x++ {
			px := float32(x)/float32(segmentsX)*width - width/2
			pz := float32(z)/float32(segmentsZ)*height - height/2

			m.Vertices = append(m.Vertices, px, 0, pz)
			m.Normals = append(m.Normals, 0, 1, 0)
			m.UVs = append(m.UVs, float32(x)/float32(segmentsX), float32(z)/float32(segmentsZ))
		}
	}

	for z := 0; z < segmentsZ; z++ {
		for x := 0; x < segmentsX; x++ {
			i0 := uint32(z*(segmentsX+1) + x)
			i1 := i0 + uint32(segmentsX) + 1

			m.Indices = append(m.Indices, i0, i1, i0+1)
			m.Indices = append(m.Indices, i1, i1+1, i0+1)
		}
	}

	return m
}

// CreateCylinder creates a cylinder mesh
func CreateCylinder(segments int, height, radius float32) *Mesh {
	m := &Mesh{}

	// Side vertices
	for i := 0; i <= segments; i++ {
		angle := float64(i) / float64(segments) * 2 * math.Pi
		x := float32(math.Cos(angle)) * radius
		z := float32(math.Sin(angle)) * radius

		// Bottom vertex
		m.Vertices = append(m.Vertices, x, -height/2, z)
		m.Normals = append(m.Normals, float32(math.Cos(angle)), 0, float32(math.Sin(angle)))
		m.UVs = append(m.UVs, float32(i)/float32(segments), 0)

		// Top vertex
		m.Vertices = append(m.Vertices, x, height/2, z)
		m.Normals = append(m.Normals, float32(math.Cos(angle)), 0, float32(math.Sin(angle)))
		m.UVs = append(m.UVs, float32(i)/float32(segments), 1)
	}

	// Side indices
	for i := 0; i < segments; i++ {
		i0 := uint32(i * 2)
		m.Indices = append(m.Indices, i0, i0+2, i0+1)
		m.Indices = append(m.Indices, i0+1, i0+2, i0+3)
	}

	return m
}

// Space deformation functions

// DeformFunc is a function that deforms a vertex
type DeformFunc func(pos emath.Vec3, time float32) emath.Vec3

// SpaceDeformer applies deformations to meshes
type SpaceDeformer struct {
	Deformations []DeformFunc
	Time         float32
}

// NewSpaceDeformer creates a new space deformer
func NewSpaceDeformer() *SpaceDeformer {
	return &SpaceDeformer{}
}

// AddDeformation adds a deformation function
func (d *SpaceDeformer) AddDeformation(fn DeformFunc) {
	d.Deformations = append(d.Deformations, fn)
}

// Update updates the deformer time
func (d *SpaceDeformer) Update(dt float32) {
	d.Time += dt
}

// DeformMesh applies all deformations to a mesh
func (d *SpaceDeformer) DeformMesh(src, dst *Mesh) {
	if len(dst.Vertices) != len(src.Vertices) {
		dst.Vertices = make([]float32, len(src.Vertices))
		dst.Normals = make([]float32, len(src.Normals))
	}
	copy(dst.Normals, src.Normals)
	copy(dst.UVs, src.UVs)
	copy(dst.Indices, src.Indices)

	vertCount := len(src.Vertices) / 3
	for i := 0; i < vertCount; i++ {
		pos := emath.Vec3{
			X: src.Vertices[i*3],
			Y: src.Vertices[i*3+1],
			Z: src.Vertices[i*3+2],
		}

		// Apply all deformations
		for _, deform := range d.Deformations {
			pos = deform(pos, d.Time)
		}

		dst.Vertices[i*3] = pos.X
		dst.Vertices[i*3+1] = pos.Y
		dst.Vertices[i*3+2] = pos.Z
	}

	// Recalculate normals
	d.recalculateNormals(dst)
}

func (d *SpaceDeformer) recalculateNormals(m *Mesh) {
	// Clear normals
	for i := range m.Normals {
		m.Normals[i] = 0
	}

	// Calculate face normals and accumulate
	for i := 0; i+2 < len(m.Indices); i += 3 {
		i0, i1, i2 := m.Indices[i], m.Indices[i+1], m.Indices[i+2]

		v0 := emath.Vec3{X: m.Vertices[i0*3], Y: m.Vertices[i0*3+1], Z: m.Vertices[i0*3+2]}
		v1 := emath.Vec3{X: m.Vertices[i1*3], Y: m.Vertices[i1*3+1], Z: m.Vertices[i1*3+2]}
		v2 := emath.Vec3{X: m.Vertices[i2*3], Y: m.Vertices[i2*3+1], Z: m.Vertices[i2*3+2]}

		edge1 := emath.Vec3{X: v1.X - v0.X, Y: v1.Y - v0.Y, Z: v1.Z - v0.Z}
		edge2 := emath.Vec3{X: v2.X - v0.X, Y: v2.Y - v0.Y, Z: v2.Z - v0.Z}

		normal := Cross(edge1, edge2)

		// Add to each vertex
		for _, idx := range []uint32{i0, i1, i2} {
			m.Normals[idx*3] += normal.X
			m.Normals[idx*3+1] += normal.Y
			m.Normals[idx*3+2] += normal.Z
		}
	}

	// Normalize
	for i := 0; i < len(m.Normals)/3; i++ {
		n := emath.Vec3{X: m.Normals[i*3], Y: m.Normals[i*3+1], Z: m.Normals[i*3+2]}
		n = Normalize3(n)
		m.Normals[i*3] = n.X
		m.Normals[i*3+1] = n.Y
		m.Normals[i*3+2] = n.Z
	}
}

// Preset deformation functions

// WaveDeform creates a wave deformation
func WaveDeform(amplitude, frequency, speed float32) DeformFunc {
	return func(pos emath.Vec3, time float32) emath.Vec3 {
		offset := float32(math.Sin(float64(pos.X*frequency+time*speed))) * amplitude
		return emath.Vec3{X: pos.X, Y: pos.Y + offset, Z: pos.Z}
	}
}

// TwistDeform creates a twist deformation around Y axis
func TwistDeform(amount float32) DeformFunc {
	return func(pos emath.Vec3, time float32) emath.Vec3 {
		angle := pos.Y * amount * time
		cos := float32(math.Cos(float64(angle)))
		sin := float32(math.Sin(float64(angle)))
		return emath.Vec3{
			X: pos.X*cos - pos.Z*sin,
			Y: pos.Y,
			Z: pos.X*sin + pos.Z*cos,
		}
	}
}

// BendDeform creates a bend deformation
func BendDeform(axis int, amount float32) DeformFunc {
	return func(pos emath.Vec3, time float32) emath.Vec3 {
		result := pos
		factor := amount * float32(math.Sin(float64(time)))

		switch axis {
		case 0: // Bend around X
			angle := pos.Z * factor
			cos := float32(math.Cos(float64(angle)))
			sin := float32(math.Sin(float64(angle)))
			result.Y = pos.Y*cos - pos.Z*sin
			result.Z = pos.Y*sin + pos.Z*cos
		case 1: // Bend around Y
			angle := pos.X * factor
			cos := float32(math.Cos(float64(angle)))
			sin := float32(math.Sin(float64(angle)))
			result.X = pos.X*cos - pos.Z*sin
			result.Z = pos.X*sin + pos.Z*cos
		}

		return result
	}
}

// PulseDeform creates a pulsing scale deformation
func PulseDeform(amount, speed float32) DeformFunc {
	return func(pos emath.Vec3, time float32) emath.Vec3 {
		scale := 1 + float32(math.Sin(float64(time*speed)))*amount
		return emath.Vec3{
			X: pos.X * scale,
			Y: pos.Y * scale,
			Z: pos.Z * scale,
		}
	}
}

// NoiseDeform creates a noise-based deformation
func NoiseDeform(amplitude, scale float32) DeformFunc {
	return func(pos emath.Vec3, time float32) emath.Vec3 {
		// Simple pseudo-noise based on position
		nx := float32(math.Sin(float64(pos.X*scale+time)*1.1 + float64(pos.Y*scale)*0.7))
		ny := float32(math.Sin(float64(pos.Y*scale+time)*1.3 + float64(pos.Z*scale)*0.9))
		nz := float32(math.Sin(float64(pos.Z*scale+time)*0.9 + float64(pos.X*scale)*1.1))

		return emath.Vec3{
			X: pos.X + nx*amplitude,
			Y: pos.Y + ny*amplitude,
			Z: pos.Z + nz*amplitude,
		}
	}
}

// SphereDeform pushes vertices toward a sphere
func SphereDeform(center emath.Vec3, radius, strength float32) DeformFunc {
	return func(pos emath.Vec3, time float32) emath.Vec3 {
		dir := emath.Vec3{
			X: pos.X - center.X,
			Y: pos.Y - center.Y,
			Z: pos.Z - center.Z,
		}
		dist := float32(math.Sqrt(float64(dir.X*dir.X + dir.Y*dir.Y + dir.Z*dir.Z)))
		if dist < 0.001 {
			return pos
		}

		// Normalize direction
		dir.X /= dist
		dir.Y /= dist
		dir.Z /= dist

		// Interpolate between current position and sphere surface
		t := strength * float32(math.Sin(float64(time)))
		targetDist := dist + (radius-dist)*t

		return emath.Vec3{
			X: center.X + dir.X*targetDist,
			Y: center.Y + dir.Y*targetDist,
			Z: center.Z + dir.Z*targetDist,
		}
	}
}

// MeltDeform creates a melting effect
func MeltDeform(meltSpeed, drip float32) DeformFunc {
	return func(pos emath.Vec3, time float32) emath.Vec3 {
		// Higher vertices melt more
		meltFactor := float32(0)
		if pos.Y > 0 {
			meltFactor = pos.Y * meltSpeed * time
		}

		// Add some dripping
		dripOffset := float32(math.Sin(float64(pos.X*drip+time)*3+float64(pos.Z*drip))) * meltFactor * 0.2

		return emath.Vec3{
			X: pos.X,
			Y: pos.Y - meltFactor + dripOffset,
			Z: pos.Z,
		}
	}
}
