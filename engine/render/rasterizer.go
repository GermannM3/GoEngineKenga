package render

import (
	"image"
	"image/color"
	"math"
	"sort"

	emath "goenginekenga/engine/math"
)

// Vertex3D represents a vertex with position, normal, UV, and color
type Vertex3D struct {
	Position emath.Vec3
	Normal   emath.Vec3
	UV       emath.Vec2
	Color    color.RGBA
}

// Triangle3D represents a triangle with three vertices
type Triangle3D struct {
	V0, V1, V2 Vertex3D
	MaterialID int
}

// Rasterizer performs software 3D rendering
type Rasterizer struct {
	Width       int
	Height      int
	ColorBuffer *image.RGBA
	DepthBuffer []float32

	// Current state
	camera       *Camera3D
	lights       []Light3D
	ambientColor color.RGBA
}

// Light3D represents a light source
type Light3D struct {
	Type      string // "directional", "point", "spot"
	Position  emath.Vec3
	Direction emath.Vec3
	Color     color.RGBA
	Intensity float32
	Range     float32 // for point/spot lights
}

// NewRasterizer creates a new software rasterizer
func NewRasterizer(width, height int) *Rasterizer {
	return &Rasterizer{
		Width:        width,
		Height:       height,
		ColorBuffer:  image.NewRGBA(image.Rect(0, 0, width, height)),
		DepthBuffer:  make([]float32, width*height),
		ambientColor: color.RGBA{R: 30, G: 30, B: 40, A: 255},
	}
}

// RenderToImage возвращает текущий цветовой буфер как *image.RGBA.
// Буфер принадлежит растеризатору, его нельзя изменять параллельно с рендерингом.
func (r *Rasterizer) RenderToImage() *image.RGBA {
	return r.ColorBuffer
}

// Resize resizes the buffers
func (r *Rasterizer) Resize(width, height int) {
	if r.Width == width && r.Height == height {
		return
	}
	r.Width = width
	r.Height = height
	r.ColorBuffer = image.NewRGBA(image.Rect(0, 0, width, height))
	r.DepthBuffer = make([]float32, width*height)
}

// Clear clears the buffers
func (r *Rasterizer) Clear(clearColor color.RGBA) {
	// Clear color buffer
	for i := 0; i < len(r.ColorBuffer.Pix); i += 4 {
		r.ColorBuffer.Pix[i] = clearColor.R
		r.ColorBuffer.Pix[i+1] = clearColor.G
		r.ColorBuffer.Pix[i+2] = clearColor.B
		r.ColorBuffer.Pix[i+3] = clearColor.A
	}
	// Clear depth buffer (set to far)
	for i := range r.DepthBuffer {
		r.DepthBuffer[i] = 1.0
	}
}

// SetCamera sets the current camera
func (r *Rasterizer) SetCamera(cam *Camera3D) {
	r.camera = cam
	if cam != nil {
		cam.SetAspectRatio(float32(r.Width) / float32(r.Height))
	}
}

// AddLight adds a light to the scene
func (r *Rasterizer) AddLight(light Light3D) {
	r.lights = append(r.lights, light)
}

// ClearLights removes all lights
func (r *Rasterizer) ClearLights() {
	r.lights = r.lights[:0]
}

// SetAmbientColor sets the ambient light color
func (r *Rasterizer) SetAmbientColor(c color.RGBA) {
	r.ambientColor = c
}

// DrawTriangle draws a single 3D triangle
func (r *Rasterizer) DrawTriangle(tri Triangle3D, modelMatrix Matrix4, texture *image.RGBA) {
	if r.camera == nil {
		return
	}

	viewProj := r.camera.GetViewProjectionMatrix()
	mvp := viewProj.Multiply(modelMatrix)

	// Transform vertices to clip space
	v0 := r.transformVertex(tri.V0, mvp, modelMatrix)
	v1 := r.transformVertex(tri.V1, mvp, modelMatrix)
	v2 := r.transformVertex(tri.V2, mvp, modelMatrix)

	// Backface culling
	if r.isBackface(v0.screenPos, v1.screenPos, v2.screenPos) {
		return
	}

	// Clip against near plane (simple check)
	if v0.clipW < 0.1 && v1.clipW < 0.1 && v2.clipW < 0.1 {
		return
	}

	// Rasterize the triangle
	r.rasterizeTriangle(v0, v1, v2, texture)
}

type transformedVertex struct {
	screenPos   emath.Vec2
	depth       float32
	clipW       float32
	worldPos    emath.Vec3
	worldNormal emath.Vec3
	uv          emath.Vec2
	color       color.RGBA
}

func (r *Rasterizer) transformVertex(v Vertex3D, mvp, model Matrix4) transformedVertex {
	// Transform to clip space
	clipPos := mvp.TransformPoint(v.Position)
	clipW := mvp[3]*v.Position.X + mvp[7]*v.Position.Y + mvp[11]*v.Position.Z + mvp[15]

	// Perspective divide and viewport transform
	var screenX, screenY float32
	if clipW != 0 {
		ndcX := clipPos.X
		ndcY := clipPos.Y
		screenX = (ndcX + 1) * 0.5 * float32(r.Width)
		screenY = (1 - ndcY) * 0.5 * float32(r.Height) // Flip Y
	}

	// Transform normal to world space
	worldNormal := Normalize3(model.TransformDirection(v.Normal))
	worldPos := model.TransformPoint(v.Position)

	return transformedVertex{
		screenPos:   emath.Vec2{X: screenX, Y: screenY},
		depth:       clipPos.Z,
		clipW:       clipW,
		worldPos:    worldPos,
		worldNormal: worldNormal,
		uv:          v.UV,
		color:       v.Color,
	}
}

func (r *Rasterizer) isBackface(v0, v1, v2 emath.Vec2) bool {
	// Calculate signed area
	area := (v1.X-v0.X)*(v2.Y-v0.Y) - (v2.X-v0.X)*(v1.Y-v0.Y)
	return area < 0
}

func (r *Rasterizer) rasterizeTriangle(v0, v1, v2 transformedVertex, texture *image.RGBA) {
	// Compute bounding box
	minX := int(math.Max(0, math.Floor(float64(min3f(v0.screenPos.X, v1.screenPos.X, v2.screenPos.X)))))
	maxX := int(math.Min(float64(r.Width-1), math.Ceil(float64(max3f(v0.screenPos.X, v1.screenPos.X, v2.screenPos.X)))))
	minY := int(math.Max(0, math.Floor(float64(min3f(v0.screenPos.Y, v1.screenPos.Y, v2.screenPos.Y)))))
	maxY := int(math.Min(float64(r.Height-1), math.Ceil(float64(max3f(v0.screenPos.Y, v1.screenPos.Y, v2.screenPos.Y)))))

	// Precompute triangle area
	area := edgeFunction(v0.screenPos, v1.screenPos, v2.screenPos)
	if area == 0 {
		return
	}

	// Rasterize
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			p := emath.Vec2{X: float32(x) + 0.5, Y: float32(y) + 0.5}

			// Barycentric coordinates
			w0 := edgeFunction(v1.screenPos, v2.screenPos, p)
			w1 := edgeFunction(v2.screenPos, v0.screenPos, p)
			w2 := edgeFunction(v0.screenPos, v1.screenPos, p)

			// Check if point is inside triangle
			if w0 >= 0 && w1 >= 0 && w2 >= 0 {
				// Normalize barycentric coordinates
				w0 /= area
				w1 /= area
				w2 /= area

				// Perspective-correct interpolation
				oneOverW := w0/v0.clipW + w1/v1.clipW + w2/v2.clipW
				if oneOverW == 0 {
					continue
				}

				// Interpolate depth
				depth := w0*v0.depth + w1*v1.depth + w2*v2.depth

				// Depth test
				idx := y*r.Width + x
				if depth >= r.DepthBuffer[idx] {
					continue
				}

				// Interpolate attributes with perspective correction
				corrW0 := (w0 / v0.clipW) / oneOverW
				corrW1 := (w1 / v1.clipW) / oneOverW
				corrW2 := (w2 / v2.clipW) / oneOverW

				// Interpolate UV
				u := corrW0*v0.uv.X + corrW1*v1.uv.X + corrW2*v2.uv.X
				v := corrW0*v0.uv.Y + corrW1*v1.uv.Y + corrW2*v2.uv.Y

				// Interpolate world position and normal for lighting
				worldPos := emath.Vec3{
					X: corrW0*v0.worldPos.X + corrW1*v1.worldPos.X + corrW2*v2.worldPos.X,
					Y: corrW0*v0.worldPos.Y + corrW1*v1.worldPos.Y + corrW2*v2.worldPos.Y,
					Z: corrW0*v0.worldPos.Z + corrW1*v1.worldPos.Z + corrW2*v2.worldPos.Z,
				}
				worldNormal := Normalize3(emath.Vec3{
					X: corrW0*v0.worldNormal.X + corrW1*v1.worldNormal.X + corrW2*v2.worldNormal.X,
					Y: corrW0*v0.worldNormal.Y + corrW1*v1.worldNormal.Y + corrW2*v2.worldNormal.Y,
					Z: corrW0*v0.worldNormal.Z + corrW1*v1.worldNormal.Z + corrW2*v2.worldNormal.Z,
				})

				// Get base color
				var baseColor color.RGBA
				if texture != nil {
					baseColor = r.sampleTexture(texture, u, v)
				} else {
					// Interpolate vertex colors
					baseColor = color.RGBA{
						R: uint8(corrW0*float32(v0.color.R) + corrW1*float32(v1.color.R) + corrW2*float32(v2.color.R)),
						G: uint8(corrW0*float32(v0.color.G) + corrW1*float32(v1.color.G) + corrW2*float32(v2.color.G)),
						B: uint8(corrW0*float32(v0.color.B) + corrW1*float32(v1.color.B) + corrW2*float32(v2.color.B)),
						A: uint8(corrW0*float32(v0.color.A) + corrW1*float32(v1.color.A) + corrW2*float32(v2.color.A)),
					}
				}

				// Calculate lighting
				finalColor := r.calculateLighting(baseColor, worldPos, worldNormal)

				// Write to buffers
				r.DepthBuffer[idx] = depth
				r.setPixel(x, y, finalColor)
			}
		}
	}
}

func (r *Rasterizer) sampleTexture(tex *image.RGBA, u, v float32) color.RGBA {
	// Wrap UV coordinates
	u = u - float32(math.Floor(float64(u)))
	v = v - float32(math.Floor(float64(v)))

	// Convert to pixel coordinates
	tx := int(u * float32(tex.Bounds().Dx()))
	ty := int(v * float32(tex.Bounds().Dy()))

	// Clamp
	if tx < 0 {
		tx = 0
	}
	if tx >= tex.Bounds().Dx() {
		tx = tex.Bounds().Dx() - 1
	}
	if ty < 0 {
		ty = 0
	}
	if ty >= tex.Bounds().Dy() {
		ty = tex.Bounds().Dy() - 1
	}

	c := tex.RGBAAt(tx, ty)
	return c
}

func (r *Rasterizer) calculateLighting(baseColor color.RGBA, worldPos, normal emath.Vec3) color.RGBA {
	// Start with ambient
	lightR := float32(r.ambientColor.R) / 255.0
	lightG := float32(r.ambientColor.G) / 255.0
	lightB := float32(r.ambientColor.B) / 255.0

	// Add contribution from each light
	for _, light := range r.lights {
		var lightDir emath.Vec3
		var attenuation float32 = 1.0

		switch light.Type {
		case "directional":
			lightDir = Normalize3(emath.Vec3{X: -light.Direction.X, Y: -light.Direction.Y, Z: -light.Direction.Z})
		case "point":
			diff := emath.Vec3{
				X: light.Position.X - worldPos.X,
				Y: light.Position.Y - worldPos.Y,
				Z: light.Position.Z - worldPos.Z,
			}
			dist := float32(math.Sqrt(float64(diff.X*diff.X + diff.Y*diff.Y + diff.Z*diff.Z)))
			if dist > light.Range {
				continue
			}
			lightDir = Normalize3(diff)
			attenuation = 1.0 - (dist / light.Range)
			attenuation *= attenuation // quadratic falloff
		default:
			continue
		}

		// Diffuse lighting (Lambert)
		ndotl := Dot(normal, lightDir)
		if ndotl < 0 {
			ndotl = 0
		}

		intensity := ndotl * light.Intensity * attenuation
		lightR += intensity * float32(light.Color.R) / 255.0
		lightG += intensity * float32(light.Color.G) / 255.0
		lightB += intensity * float32(light.Color.B) / 255.0
	}

	// Clamp lighting
	if lightR > 1 {
		lightR = 1
	}
	if lightG > 1 {
		lightG = 1
	}
	if lightB > 1 {
		lightB = 1
	}

	// Apply lighting to base color
	return color.RGBA{
		R: uint8(float32(baseColor.R) * lightR),
		G: uint8(float32(baseColor.G) * lightG),
		B: uint8(float32(baseColor.B) * lightB),
		A: baseColor.A,
	}
}

func (r *Rasterizer) setPixel(x, y int, c color.RGBA) {
	if x < 0 || x >= r.Width || y < 0 || y >= r.Height {
		return
	}
	idx := (y*r.Width + x) * 4
	r.ColorBuffer.Pix[idx] = c.R
	r.ColorBuffer.Pix[idx+1] = c.G
	r.ColorBuffer.Pix[idx+2] = c.B
	r.ColorBuffer.Pix[idx+3] = c.A
}

// DrawMesh draws a mesh with transformation
func (r *Rasterizer) DrawMesh(positions []float32, indices []uint32, normals []float32, uvs []float32, modelMatrix Matrix4, texture *image.RGBA, vertexColor color.RGBA) {
	// Build triangles from mesh data
	for i := 0; i+2 < len(indices); i += 3 {
		i0, i1, i2 := indices[i], indices[i+1], indices[i+2]

		tri := Triangle3D{
			V0: r.buildVertex(positions, normals, uvs, int(i0), vertexColor),
			V1: r.buildVertex(positions, normals, uvs, int(i1), vertexColor),
			V2: r.buildVertex(positions, normals, uvs, int(i2), vertexColor),
		}

		r.DrawTriangle(tri, modelMatrix, texture)
	}
}

func (r *Rasterizer) buildVertex(positions, normals, uvs []float32, idx int, defaultColor color.RGBA) Vertex3D {
	v := Vertex3D{Color: defaultColor}

	// Position
	if idx*3+2 < len(positions) {
		v.Position = emath.Vec3{
			X: positions[idx*3],
			Y: positions[idx*3+1],
			Z: positions[idx*3+2],
		}
	}

	// Normal
	if normals != nil && idx*3+2 < len(normals) {
		v.Normal = emath.Vec3{
			X: normals[idx*3],
			Y: normals[idx*3+1],
			Z: normals[idx*3+2],
		}
	} else {
		v.Normal = emath.Vec3{Y: 1} // Default up
	}

	// UV
	if uvs != nil && idx*2+1 < len(uvs) {
		v.UV = emath.Vec2{
			X: uvs[idx*2],
			Y: uvs[idx*2+1],
		}
	}

	return v
}

// SortTrianglesByDepth sorts triangles back-to-front for transparency
func SortTrianglesByDepth(triangles []Triangle3D, camera *Camera3D) {
	if camera == nil {
		return
	}
	camPos := camera.Position

	sort.Slice(triangles, func(i, j int) bool {
		// Calculate center of each triangle
		ci := triangleCenter(triangles[i])
		cj := triangleCenter(triangles[j])

		// Distance from camera
		di := distanceSquared(ci, camPos)
		dj := distanceSquared(cj, camPos)

		return di > dj // Back to front
	})
}

func triangleCenter(t Triangle3D) emath.Vec3 {
	return emath.Vec3{
		X: (t.V0.Position.X + t.V1.Position.X + t.V2.Position.X) / 3,
		Y: (t.V0.Position.Y + t.V1.Position.Y + t.V2.Position.Y) / 3,
		Z: (t.V0.Position.Z + t.V1.Position.Z + t.V2.Position.Z) / 3,
	}
}

func distanceSquared(a, b emath.Vec3) float32 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return dx*dx + dy*dy + dz*dz
}

// Helper functions
func edgeFunction(a, b, c emath.Vec2) float32 {
	return (c.X-a.X)*(b.Y-a.Y) - (c.Y-a.Y)*(b.X-a.X)
}

func min3f(a, b, c float32) float32 {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func max3f(a, b, c float32) float32 {
	if a > b {
		if a > c {
			return a
		}
		return c
	}
	if b > c {
		return b
	}
	return c
}
