package physics

import (
	"math"

	emath "goenginekenga/engine/math"
)

// WaterSurface represents a dynamic water surface
type WaterSurface struct {
	Width      int
	Height     int
	CellSize   float32
	Heights    []float32 // Current heights
	Velocities []float32 // Vertical velocities

	// Water properties
	Damping   float32 // Wave damping (0-1)
	Tension   float32 // Surface tension
	WaveSpeed float32 // Wave propagation speed

	// Position in world
	Position emath.Vec3
}

// NewWaterSurface creates a new water surface
func NewWaterSurface(width, height int, cellSize float32) *WaterSurface {
	size := width * height
	return &WaterSurface{
		Width:      width,
		Height:     height,
		CellSize:   cellSize,
		Heights:    make([]float32, size),
		Velocities: make([]float32, size),
		Damping:    0.98,
		Tension:    0.025,
		WaveSpeed:  2.0,
	}
}

// Update simulates the water surface
func (w *WaterSurface) Update(dt float32) {
	// Wave equation simulation using finite differences
	newHeights := make([]float32, len(w.Heights))
	copy(newHeights, w.Heights)

	for y := 1; y < w.Height-1; y++ {
		for x := 1; x < w.Width-1; x++ {
			idx := y*w.Width + x

			// Get neighbor heights
			left := w.Heights[idx-1]
			right := w.Heights[idx+1]
			up := w.Heights[idx-w.Width]
			down := w.Heights[idx+w.Width]
			center := w.Heights[idx]

			// Calculate laplacian (wave propagation)
			laplacian := (left+right+up+down)/4.0 - center

			// Update velocity with wave equation
			w.Velocities[idx] += laplacian * w.Tension * w.WaveSpeed * dt

			// Apply damping
			w.Velocities[idx] *= w.Damping

			// Update height
			newHeights[idx] += w.Velocities[idx]
		}
	}

	copy(w.Heights, newHeights)
}

// Disturb creates a disturbance at a point
func (w *WaterSurface) Disturb(worldX, worldZ, force, radius float32) {
	// Convert world coordinates to grid coordinates
	localX := (worldX - w.Position.X) / w.CellSize
	localZ := (worldZ - w.Position.Z) / w.CellSize

	gridRadius := radius / w.CellSize

	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			dx := float32(x) - localX
			dy := float32(y) - localZ
			dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))

			if dist < gridRadius {
				// Smooth falloff
				falloff := 1.0 - dist/gridRadius
				falloff = falloff * falloff
				idx := y*w.Width + x
				w.Heights[idx] += force * falloff
			}
		}
	}
}

// GetHeightAt returns the interpolated height at world coordinates
func (w *WaterSurface) GetHeightAt(worldX, worldZ float32) float32 {
	localX := (worldX - w.Position.X) / w.CellSize
	localZ := (worldZ - w.Position.Z) / w.CellSize

	if localX < 0 || localX >= float32(w.Width-1) || localZ < 0 || localZ >= float32(w.Height-1) {
		return w.Position.Y
	}

	// Bilinear interpolation
	x0 := int(localX)
	z0 := int(localZ)
	fx := localX - float32(x0)
	fz := localZ - float32(z0)

	h00 := w.Heights[z0*w.Width+x0]
	h10 := w.Heights[z0*w.Width+x0+1]
	h01 := w.Heights[(z0+1)*w.Width+x0]
	h11 := w.Heights[(z0+1)*w.Width+x0+1]

	h0 := h00*(1-fx) + h10*fx
	h1 := h01*(1-fx) + h11*fx

	return w.Position.Y + h0*(1-fz) + h1*fz
}

// GetNormalAt returns the surface normal at world coordinates
func (w *WaterSurface) GetNormalAt(worldX, worldZ float32) emath.Vec3 {
	delta := w.CellSize * 0.5

	h0 := w.GetHeightAt(worldX-delta, worldZ)
	h1 := w.GetHeightAt(worldX+delta, worldZ)
	h2 := w.GetHeightAt(worldX, worldZ-delta)
	h3 := w.GetHeightAt(worldX, worldZ+delta)

	// Calculate normal from height differences
	normal := emath.Vec3{
		X: h0 - h1,
		Y: 2 * delta,
		Z: h2 - h3,
	}

	// Normalize
	len := float32(math.Sqrt(float64(normal.X*normal.X + normal.Y*normal.Y + normal.Z*normal.Z)))
	if len > 0 {
		normal.X /= len
		normal.Y /= len
		normal.Z /= len
	}

	return normal
}

// Buoyancy represents buoyancy physics
type Buoyancy struct {
	WaterDensity    float32 // kg/m³ (1000 for water)
	DragCoefficient float32
}

// NewBuoyancy creates a new buoyancy system
func NewBuoyancy() *Buoyancy {
	return &Buoyancy{
		WaterDensity:    1000,
		DragCoefficient: 0.5,
	}
}

// CalculateBuoyancyForce calculates the buoyancy force on an object
func (b *Buoyancy) CalculateBuoyancyForce(
	objectPos emath.Vec3,
	objectVolume float32,
	waterLevel float32,
	objectBottom, objectTop float32,
) emath.Vec3 {
	// Calculate submerged fraction
	submergedDepth := waterLevel - objectBottom
	if submergedDepth <= 0 {
		return emath.Vec3{} // Above water
	}

	objectHeight := objectTop - objectBottom
	if objectHeight <= 0 {
		return emath.Vec3{}
	}

	submergedFraction := submergedDepth / objectHeight
	if submergedFraction > 1 {
		submergedFraction = 1
	}

	// Buoyancy force = water density * gravity * displaced volume
	displacedVolume := objectVolume * submergedFraction
	buoyancyMagnitude := b.WaterDensity * 9.81 * displacedVolume

	return emath.Vec3{Y: buoyancyMagnitude}
}

// CalculateWaterDrag calculates drag force from water
func (b *Buoyancy) CalculateWaterDrag(
	velocity emath.Vec3,
	submergedArea float32,
) emath.Vec3 {
	// Drag = 0.5 * density * velocity² * dragCoefficient * area
	speed := float32(math.Sqrt(float64(velocity.X*velocity.X + velocity.Y*velocity.Y + velocity.Z*velocity.Z)))
	if speed < 0.001 {
		return emath.Vec3{}
	}

	dragMagnitude := 0.5 * b.WaterDensity * speed * speed * b.DragCoefficient * submergedArea

	// Direction opposite to velocity
	return emath.Vec3{
		X: -velocity.X / speed * dragMagnitude,
		Y: -velocity.Y / speed * dragMagnitude,
		Z: -velocity.Z / speed * dragMagnitude,
	}
}

// OceanWaves generates ocean wave heights using Gerstner waves
type OceanWaves struct {
	Waves []GerstnerWave
	Time  float32
}

// GerstnerWave represents a single Gerstner wave component
type GerstnerWave struct {
	Direction  emath.Vec2 // Normalized wave direction
	Wavelength float32
	Amplitude  float32
	Steepness  float32 // 0-1, how sharp the wave peaks
	Speed      float32
}

// NewOceanWaves creates a new ocean wave system
func NewOceanWaves() *OceanWaves {
	o := &OceanWaves{}

	// Add some default waves
	o.AddWave(emath.Vec2{X: 1, Y: 0}, 50, 1.0, 0.5, 5)
	o.AddWave(emath.Vec2{X: 0.7, Y: 0.7}, 30, 0.5, 0.3, 4)
	o.AddWave(emath.Vec2{X: 0.3, Y: 0.9}, 20, 0.3, 0.2, 3)
	o.AddWave(emath.Vec2{X: -0.5, Y: 0.5}, 15, 0.2, 0.1, 2)

	return o
}

// AddWave adds a wave component
func (o *OceanWaves) AddWave(direction emath.Vec2, wavelength, amplitude, steepness, speed float32) {
	// Normalize direction
	len := float32(math.Sqrt(float64(direction.X*direction.X + direction.Y*direction.Y)))
	if len > 0 {
		direction.X /= len
		direction.Y /= len
	}

	o.Waves = append(o.Waves, GerstnerWave{
		Direction:  direction,
		Wavelength: wavelength,
		Amplitude:  amplitude,
		Steepness:  steepness,
		Speed:      speed,
	})
}

// Update updates the wave time
func (o *OceanWaves) Update(dt float32) {
	o.Time += dt
}

// GetDisplacement calculates the displacement at a point
func (o *OceanWaves) GetDisplacement(x, z float32) emath.Vec3 {
	displacement := emath.Vec3{}

	for _, wave := range o.Waves {
		k := 2 * math.Pi / wave.Wavelength
		phase := float32(k)*(wave.Direction.X*x+wave.Direction.Y*z) - float32(k)*wave.Speed*o.Time

		sinPhase := float32(math.Sin(float64(phase)))
		cosPhase := float32(math.Cos(float64(phase)))

		// Gerstner wave formula
		qi := wave.Steepness / (float32(k) * wave.Amplitude * float32(len(o.Waves)))

		displacement.X += qi * wave.Amplitude * wave.Direction.X * cosPhase
		displacement.Y += wave.Amplitude * sinPhase
		displacement.Z += qi * wave.Amplitude * wave.Direction.Y * cosPhase
	}

	return displacement
}

// GetHeightAt returns the wave height at a position
func (o *OceanWaves) GetHeightAt(x, z float32) float32 {
	disp := o.GetDisplacement(x, z)
	return disp.Y
}

// GetNormalAt calculates the surface normal
func (o *OceanWaves) GetNormalAt(x, z float32) emath.Vec3 {
	delta := float32(0.1)

	h0 := o.GetHeightAt(x-delta, z)
	h1 := o.GetHeightAt(x+delta, z)
	h2 := o.GetHeightAt(x, z-delta)
	h3 := o.GetHeightAt(x, z+delta)

	normal := emath.Vec3{
		X: h0 - h1,
		Y: 2 * delta,
		Z: h2 - h3,
	}

	len := float32(math.Sqrt(float64(normal.X*normal.X + normal.Y*normal.Y + normal.Z*normal.Z)))
	if len > 0 {
		normal.X /= len
		normal.Y /= len
		normal.Z /= len
	}

	return normal
}

// Ship physics for pirate game

// Ship represents a ship with water physics
type Ship struct {
	Position   emath.Vec3
	Rotation   emath.Vec3 // Euler angles
	Velocity   emath.Vec3
	AngularVel emath.Vec3

	Mass       float32
	Length     float32
	Width      float32
	Height     float32
	DraftDepth float32 // How deep the hull sits

	// Sail
	SailArea  float32
	SailAngle float32 // Relative to ship

	// Rudder
	RudderAngle float32 // -1 to 1
}

// NewShip creates a new ship
func NewShip(pos emath.Vec3) *Ship {
	return &Ship{
		Position:   pos,
		Mass:       5000, // kg
		Length:     15,
		Width:      5,
		Height:     3,
		DraftDepth: 1.5,
		SailArea:   50,
	}
}

// UpdateShipPhysics updates ship physics with water interaction
func (s *Ship) Update(dt float32, ocean *OceanWaves, wind emath.Vec2) {
	// Get water height and normal at ship position
	waterHeight := ocean.GetHeightAt(s.Position.X, s.Position.Z)
	waterNormal := ocean.GetNormalAt(s.Position.X, s.Position.Z)

	// Buoyancy
	submerged := s.Position.Y - s.DraftDepth - waterHeight
	if submerged < 0 {
		buoyancy := -submerged * 9.81 * 1000 * s.Length * s.Width / s.Mass
		s.Velocity.Y += buoyancy * dt
	}

	// Gravity
	s.Velocity.Y -= 9.81 * dt

	// Water drag
	waterDrag := float32(0.1)
	s.Velocity.X *= (1 - waterDrag*dt)
	s.Velocity.Z *= (1 - waterDrag*dt)

	// Wind on sails
	if s.SailArea > 0 {
		// Calculate effective wind angle
		shipForward := emath.Vec2{
			X: float32(math.Sin(float64(s.Rotation.Y * math.Pi / 180))),
			Y: float32(math.Cos(float64(s.Rotation.Y * math.Pi / 180))),
		}

		windSpeed := float32(math.Sqrt(float64(wind.X*wind.X + wind.Y*wind.Y)))
		if windSpeed > 0 {
			windDir := emath.Vec2{X: wind.X / windSpeed, Y: wind.Y / windSpeed}

			// Dot product for wind effectiveness
			windDot := shipForward.X*windDir.X + shipForward.Y*windDir.Y

			// Can't sail directly into wind
			if windDot > -0.5 {
				sailForce := s.SailArea * windSpeed * 0.001 * (1 + windDot)
				s.Velocity.X += shipForward.X * sailForce * dt
				s.Velocity.Z += shipForward.Y * sailForce * dt
			}
		}
	}

	// Rudder steering
	if s.RudderAngle != 0 {
		speed := float32(math.Sqrt(float64(s.Velocity.X*s.Velocity.X + s.Velocity.Z*s.Velocity.Z)))
		turnRate := s.RudderAngle * speed * 0.5 // Faster speed = faster turn
		s.Rotation.Y += turnRate * dt * 180 / math.Pi
	}

	// Apply rotation from water surface
	s.Rotation.X = float32(math.Atan2(float64(waterNormal.Z), float64(waterNormal.Y))) * 180 / math.Pi * 0.5
	s.Rotation.Z = float32(math.Atan2(float64(waterNormal.X), float64(waterNormal.Y))) * 180 / math.Pi * 0.5

	// Update position
	s.Position.X += s.Velocity.X * dt
	s.Position.Y += s.Velocity.Y * dt
	s.Position.Z += s.Velocity.Z * dt

	// Keep ship at water level
	targetY := waterHeight + s.DraftDepth
	s.Position.Y = s.Position.Y*0.9 + targetY*0.1
}

// SetRudder sets the rudder angle (-1 to 1)
func (s *Ship) SetRudder(angle float32) {
	if angle < -1 {
		angle = -1
	}
	if angle > 1 {
		angle = 1
	}
	s.RudderAngle = angle
}
