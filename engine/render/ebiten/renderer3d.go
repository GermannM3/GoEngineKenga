package ebiten

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/particles"
	"goenginekenga/engine/render"
)

// Renderer3D handles 3D rendering using software rasterization
type Renderer3D struct {
	rasterizer *render.Rasterizer
	camera     *render.Camera3D
	lights     []render.Light3D

	// Particle system rendering
	particleSystems []*particles.System

	// Output image for Ebiten
	outputImage *ebiten.Image

	width, height int
}

// NewRenderer3D creates a new 3D renderer
func NewRenderer3D(width, height int) *Renderer3D {
	r := &Renderer3D{
		rasterizer: render.NewRasterizer(width, height),
		camera:     render.NewCamera3D(),
		width:      width,
		height:     height,
	}

	// Default camera position
	r.camera.SetPosition(emath.Vec3{X: 0, Y: 5, Z: 10})
	r.camera.SetTarget(emath.Vec3{X: 0, Y: 0, Z: 0})

	// Default lights
	r.AddLight(render.Light3D{
		Type:      "directional",
		Direction: emath.Vec3{X: -0.5, Y: -1, Z: -0.3},
		Color:     color.RGBA{R: 255, G: 250, B: 240, A: 255},
		Intensity: 1.0,
	})

	return r
}

// RenderToImage реализует интерфейс render.FrameRenderer и
// предоставляет доступ к последнему буферу кадра.
func (r *Renderer3D) RenderToImage() *image.RGBA {
	if r == nil || r.rasterizer == nil {
		return nil
	}
	return r.rasterizer.RenderToImage()
}

// RenderToBuffer реализует интерфейс render.FrameRenderer и
// предоставляет доступ к последнему буферу кадра в формате RGBA байтов.
func (r *Renderer3D) RenderToBuffer() ([]byte, int, int, error) {
	if r == nil || r.rasterizer == nil {
		return nil, 0, 0, nil
	}
	return r.rasterizer.RenderToBuffer()
}

// Resize resizes the renderer
func (r *Renderer3D) Resize(width, height int) {
	if r.width == width && r.height == height {
		return
	}
	r.width = width
	r.height = height
	r.rasterizer.Resize(width, height)
	r.outputImage = nil // Will be recreated on next render
}

// GetCamera returns the camera
func (r *Renderer3D) GetCamera() *render.Camera3D {
	return r.camera
}

// AddLight adds a light
func (r *Renderer3D) AddLight(light render.Light3D) {
	r.lights = append(r.lights, light)
}

// ClearLights clears all lights
func (r *Renderer3D) ClearLights() {
	r.lights = r.lights[:0]
}

// AddParticleSystem adds a particle system for rendering
func (r *Renderer3D) AddParticleSystem(ps *particles.System) {
	r.particleSystems = append(r.particleSystems, ps)
}

// RenderWorld renders the entire world
func (r *Renderer3D) RenderWorld(world *ecs.World, resolver *asset.Resolver, clearColor color.RGBA) *ebiten.Image {
	// Clear buffers
	r.rasterizer.Clear(clearColor)

	// Setup camera
	r.rasterizer.SetCamera(r.camera)

	// Setup lights
	r.rasterizer.ClearLights()
	r.rasterizer.SetAmbientColor(color.RGBA{R: 40, G: 45, B: 55, A: 255})
	for _, light := range r.lights {
		r.rasterizer.AddLight(light)
	}

	// Update camera from world if there's a camera entity
	if world != nil {
		r.updateCameraFromWorld(world)
		r.updateLightsFromWorld(world)
		r.renderEntities(world, resolver)
		r.renderTrajectories(world)
	}

	// Draw world-space helpers (grid + axes), чтобы окно никогда не было пустым.
	r.renderGridAndAxes()

	// Render particles
	r.renderParticles()

	// Create Ebiten image from rasterizer output
	return r.createOutputImage()
}

func (r *Renderer3D) updateCameraFromWorld(world *ecs.World) {
	for _, id := range world.Entities() {
		cam, hasCam := world.GetCamera(id)
		if !hasCam {
			continue
		}

		tr, hasTr := world.GetTransform(id)
		if !hasTr {
			continue
		}

		// Set camera position from transform
		r.camera.SetPosition(tr.Position)

		// Calculate target from rotation
		// Simple: assume camera looks forward along Z axis, rotated by Y
		radY := tr.Rotation.Y * math.Pi / 180
		forward := emath.Vec3{
			X: float32(math.Sin(float64(radY))),
			Y: 0,
			Z: float32(math.Cos(float64(radY))),
		}
		r.camera.SetTarget(emath.Vec3{
			X: tr.Position.X + forward.X*10,
			Y: tr.Position.Y,
			Z: tr.Position.Z + forward.Z*10,
		})

		// Set FOV
		if cam.FovYDegrees > 0 {
			r.camera.SetFOV(cam.FovYDegrees)
		}

		break // Only use first camera
	}
}

func (r *Renderer3D) updateLightsFromWorld(world *ecs.World) {
	// Clear existing world lights (keep default)
	worldLights := []render.Light3D{}

	for _, id := range world.Entities() {
		light, hasLight := world.GetLight(id)
		if !hasLight {
			continue
		}

		tr, hasTr := world.GetTransform(id)

		// Convert ECS Light to render Light3D
		lightColor := color.RGBA{R: light.ColorR, G: light.ColorG, B: light.ColorB, A: 255}
		if lightColor.R == 0 && lightColor.G == 0 && lightColor.B == 0 {
			// Use ColorRGB if RGBA not set
			lightColor = color.RGBA{
				R: uint8(light.ColorRGB.X * 255),
				G: uint8(light.ColorRGB.Y * 255),
				B: uint8(light.ColorRGB.Z * 255),
				A: 255,
			}
		}

		l := render.Light3D{
			Type:      light.Kind,
			Color:     lightColor,
			Intensity: light.Intensity,
			Range:     light.Range,
		}

		if hasTr {
			l.Position = tr.Position
			// Calculate direction from rotation
			radX := tr.Rotation.X * math.Pi / 180
			radY := tr.Rotation.Y * math.Pi / 180
			l.Direction = emath.Vec3{
				X: float32(math.Sin(float64(radY)) * math.Cos(float64(radX))),
				Y: float32(-math.Sin(float64(radX))),
				Z: float32(math.Cos(float64(radY)) * math.Cos(float64(radX))),
			}
		}

		worldLights = append(worldLights, l)
	}

	// Add world lights to rasterizer
	for _, l := range worldLights {
		r.rasterizer.AddLight(l)
	}
}

func (r *Renderer3D) renderEntities(world *ecs.World, resolver *asset.Resolver) {
	for _, id := range world.Entities() {
		mr, hasMR := world.GetMeshRenderer(id)
		if !hasMR {
			continue
		}

		tr, hasTr := world.GetTransform(id)
		if !hasTr {
			tr = ecs.Transform{
				Position: emath.Vec3{},
				Rotation: emath.Vec3{},
				Scale:    emath.Vec3{X: 1, Y: 1, Z: 1},
			}
		}

		// Build model matrix
		modelMatrix := r.buildModelMatrix(&tr)

		// Try to load mesh from resolver
		var positions, normals, uvs []float32
		var indices []uint32
		var texture *image.RGBA

		if resolver != nil && mr.MeshAssetID != "" {
			// Try to get mesh asset via resolver
			if meshAsset, err := resolver.ResolveMeshByAssetID(mr.MeshAssetID); err == nil {
				positions = meshAsset.Positions
				normals = meshAsset.Normals
				uvs = meshAsset.UV0
				indices = meshAsset.Indices
			}
		}

		// If no mesh found, use a default cube
		if len(positions) == 0 {
			cube := render.CreateCube()
			positions = cube.Vertices
			normals = cube.Normals
			uvs = cube.UVs
			indices = cube.Indices
		}

		// Draw mesh
		vertexColor := color.RGBA{R: 200, G: 200, B: 200, A: 255}
		if mr.ColorA > 0 {
			vertexColor = color.RGBA{R: mr.ColorR, G: mr.ColorG, B: mr.ColorB, A: mr.ColorA}
		}

		r.rasterizer.DrawMesh(positions, indices, normals, uvs, modelMatrix, texture, vertexColor)
	}
}

func (r *Renderer3D) buildModelMatrix(tr *ecs.Transform) render.Matrix4 {
	// Translation
	t := render.Translate(tr.Position)

	// Rotation (ZYX order)
	rx := render.RotateX(tr.Rotation.X)
	ry := render.RotateY(tr.Rotation.Y)
	rz := render.RotateZ(tr.Rotation.Z)

	// Scale - ensure non-zero
	scale := tr.Scale
	if scale.X == 0 {
		scale.X = 1
	}
	if scale.Y == 0 {
		scale.Y = 1
	}
	if scale.Z == 0 {
		scale.Z = 1
	}
	s := render.Scale(scale)

	// Combine: T * Rz * Ry * Rx * S
	return t.Multiply(rz.Multiply(ry.Multiply(rx.Multiply(s))))
}

func (r *Renderer3D) renderParticles() {
	if r.camera == nil {
		return
	}

	viewProj := r.camera.GetViewProjectionMatrix()

	for _, ps := range r.particleSystems {
		for i := range ps.Particles {
			p := &ps.Particles[i]
			if p.Life <= 0 {
				continue
			}

			// Project particle position to screen
			clipPos := viewProj.TransformPoint(p.Position)
			clipW := viewProj[3]*p.Position.X + viewProj[7]*p.Position.Y + viewProj[11]*p.Position.Z + viewProj[15]

			if clipW <= 0.1 {
				continue // Behind camera
			}

			screenX := int((clipPos.X + 1) * 0.5 * float32(r.width))
			screenY := int((1 - clipPos.Y) * 0.5 * float32(r.height))

			// Draw particle as a filled circle
			size := int(p.Size * 10 / clipW) // Perspective size
			if size < 1 {
				size = 1
			}

			// Get emitter for color interpolation
			var particleColor color.RGBA = p.Color
			for _, emitter := range ps.Emitters {
				particleColor = emitter.GetParticleColor(p.Life)
				break
			}

			// Draw filled circle
			for dy := -size; dy <= size; dy++ {
				for dx := -size; dx <= size; dx++ {
					if dx*dx+dy*dy <= size*size {
						px := screenX + dx
						py := screenY + dy
						if px >= 0 && px < r.width && py >= 0 && py < r.height {
							idx := (py*r.width + px) * 4
							// Blend with alpha
							alpha := float32(particleColor.A) / 255.0 * p.Life
							r.rasterizer.ColorBuffer.Pix[idx] = uint8(float32(r.rasterizer.ColorBuffer.Pix[idx])*(1-alpha) + float32(particleColor.R)*alpha)
							r.rasterizer.ColorBuffer.Pix[idx+1] = uint8(float32(r.rasterizer.ColorBuffer.Pix[idx+1])*(1-alpha) + float32(particleColor.G)*alpha)
							r.rasterizer.ColorBuffer.Pix[idx+2] = uint8(float32(r.rasterizer.ColorBuffer.Pix[idx+2])*(1-alpha) + float32(particleColor.B)*alpha)
							r.rasterizer.ColorBuffer.Pix[idx+3] = 255
						}
					}
				}
			}
		}
	}
}

// renderTrajectories рисует Trajectory-компоненты как 2D-линии поверх 3D-сцены.
func (r *Renderer3D) renderTrajectories(world *ecs.World) {
	if world == nil || r.camera == nil || r.rasterizer == nil || r.rasterizer.ColorBuffer == nil {
		return
	}
	viewProj := r.camera.GetViewProjectionMatrix()

	for _, id := range world.Entities() {
		traj, ok := world.GetTrajectory(id)
		if !ok || len(traj.Points) < 2 {
			continue
		}

		col := traj.Color
		if col.A == 0 {
			col = color.RGBA{R: 255, G: 200, B: 80, A: 255}
		}
		thickness := int(traj.Width)
		if thickness <= 0 {
			thickness = 2
		}

		var lastScreenX, lastScreenY int
		var hasLast bool

		for _, p := range traj.Points {
			clip := viewProj.TransformPoint(p)
			w := viewProj[3]*p.X + viewProj[7]*p.Y + viewProj[11]*p.Z + viewProj[15]
			if w <= 0.1 {
				hasLast = false
				continue
			}

			sx := int((clip.X + 1) * 0.5 * float32(r.width))
			sy := int((1 - clip.Y) * 0.5 * float32(r.height))

			if hasLast {
				r.drawLine2D(lastScreenX, lastScreenY, sx, sy, col, thickness)
			}
			lastScreenX, lastScreenY = sx, sy
			hasLast = true
		}
	}
}

// renderGridAndAxes рисует простую сетку по XZ-плоскости (y=0) и оси XYZ,
// чтобы даже при пустой сцене окно не выглядело полностью чёрным.
func (r *Renderer3D) renderGridAndAxes() {
	if r.camera == nil || r.rasterizer == nil || r.rasterizer.ColorBuffer == nil {
		return
	}

	viewProj := r.camera.GetViewProjectionMatrix()

	project := func(p emath.Vec3) (int, int, bool) {
		clip := viewProj.TransformPoint(p)
		w := viewProj[3]*p.X + viewProj[7]*p.Y + viewProj[11]*p.Z + viewProj[15]
		if w <= 0.1 {
			return 0, 0, false
		}
		sx := int((clip.X + 1) * 0.5 * float32(r.width))
		sy := int((1 - clip.Y) * 0.5 * float32(r.height))
		return sx, sy, true
	}

	// Сетка на плоскости XZ
	gridColor := color.RGBA{R: 60, G: 70, B: 90, A: 255}
	origin := emath.V3(0, 0, 0)
	extent := float32(10)
	step := float32(1)

	for x := -extent; x <= extent; x += step {
		p0 := emath.V3(x, 0, -extent)
		p1 := emath.V3(x, 0, extent)
		x0, y0, ok0 := project(p0)
		x1, y1, ok1 := project(p1)
		if ok0 && ok1 {
			r.drawLine2D(x0, y0, x1, y1, gridColor, 1)
		}
	}
	for z := -extent; z <= extent; z += step {
		p0 := emath.V3(-extent, 0, z)
		p1 := emath.V3(extent, 0, z)
		x0, y0, ok0 := project(p0)
		x1, y1, ok1 := project(p1)
		if ok0 && ok1 {
			r.drawLine2D(x0, y0, x1, y1, gridColor, 1)
		}
	}

	// Оси координат
	axisLen := float32(2.5)
	axes := []struct {
		from, to emath.Vec3
		col      color.RGBA
	}{
		{origin, emath.V3(axisLen, 0, 0), color.RGBA{R: 220, G: 80, B: 80, A: 255}},   // X — красный
		{origin, emath.V3(0, axisLen, 0), color.RGBA{R: 80, G: 220, B: 80, A: 255}},   // Y — зелёный
		{origin, emath.V3(0, 0, axisLen), color.RGBA{R: 80, G: 160, B: 240, A: 255}}, // Z — синий
	}

	for _, a := range axes {
		x0, y0, ok0 := project(a.from)
		x1, y1, ok1 := project(a.to)
		if ok0 && ok1 {
			r.drawLine2D(x0, y0, x1, y1, a.col, 3)
		}
	}
}

func (r *Renderer3D) drawLine2D(x0, y0, x1, y1 int, col color.RGBA, thickness int) {
	if r.rasterizer == nil || r.rasterizer.ColorBuffer == nil {
		return
	}

	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)

	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}

	err := dx + dy

	for {
		r.plotThickPixel(x0, y0, col, thickness)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (r *Renderer3D) plotThickPixel(x, y int, col color.RGBA, thickness int) {
	if thickness < 1 {
		thickness = 1
	}
	radius := thickness / 2
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy > radius*radius {
				continue
			}
			px := x + dx
			py := y + dy
			if px < 0 || py < 0 || px >= r.width || py >= r.height {
				continue
			}
			idx := (py*r.width + px) * 4
			buf := r.rasterizer.ColorBuffer.Pix
			if idx+3 >= len(buf) {
				continue
			}
			alpha := float32(col.A) / 255.0
			if alpha <= 0 {
				continue
			}
			inv := 1 - alpha
			buf[idx] = uint8(float32(buf[idx])*inv + float32(col.R)*alpha)
			buf[idx+1] = uint8(float32(buf[idx+1])*inv + float32(col.G)*alpha)
			buf[idx+2] = uint8(float32(buf[idx+2])*inv + float32(col.B)*alpha)
			buf[idx+3] = 255
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (r *Renderer3D) createOutputImage() *ebiten.Image {
	if r.outputImage == nil || r.outputImage.Bounds().Dx() != r.width {
		r.outputImage = ebiten.NewImage(r.width, r.height)
	}

	// Copy rasterizer buffer to Ebiten image
	r.outputImage.WritePixels(r.rasterizer.ColorBuffer.Pix)

	return r.outputImage
}

// DrawToScreen draws the rendered 3D scene to an Ebiten screen
func (r *Renderer3D) DrawToScreen(screen *ebiten.Image, world *ecs.World, resolver *asset.Resolver, clearColor color.RGBA) {
	w, h := screen.Size()
	r.Resize(w, h)

	output := r.RenderWorld(world, resolver, clearColor)

	screen.DrawImage(output, nil)
}

// UpdateParticleSystems updates all particle systems
func (r *Renderer3D) UpdateParticleSystems(dt float32) {
	for _, ps := range r.particleSystems {
		ps.Update(dt)
	}
}
