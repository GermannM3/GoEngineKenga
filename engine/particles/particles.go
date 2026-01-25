package particles

import (
	"image/color"
	"math"
	"math/rand"

	emath "goenginekenga/engine/math"
)

// Particle represents a single particle
type Particle struct {
	Position emath.Vec3
	Velocity emath.Vec3
	Color    color.RGBA
	Size     float32
	Life     float32 // Remaining life (0-1)
	MaxLife  float32 // Total lifetime in seconds
	Rotation float32
	RotSpeed float32
}

// Emitter defines how particles are spawned
type Emitter struct {
	Position  emath.Vec3
	Direction emath.Vec3 // Main emission direction
	Spread    float32    // Cone spread in degrees (0 = line, 180 = hemisphere)

	// Emission settings
	Rate          float32 // Particles per second
	Burst         int     // Particles to spawn immediately
	BurstInterval float32 // Time between bursts (0 = no repeat)

	// Particle properties
	InitialSpeed Range
	InitialSize  Range
	Lifetime     Range
	InitialColor color.RGBA
	EndColor     color.RGBA

	// Physics
	Gravity emath.Vec3
	Drag    float32

	// Size over lifetime
	SizeOverLife Curve

	// Rotation
	InitialRotation Range
	RotationSpeed   Range

	// Internal state
	accumulator float32
	burstTimer  float32
}

// Range represents a min-max range for randomization
type Range struct {
	Min, Max float32
}

// Curve represents a value curve over lifetime
type Curve struct {
	Points []CurvePoint
}

// CurvePoint is a point on a curve
type CurvePoint struct {
	Time  float32 // 0-1
	Value float32
}

// System manages particles
type System struct {
	Particles    []Particle
	Emitters     []*Emitter
	MaxParticles int
}

// NewSystem creates a new particle system
func NewSystem(maxParticles int) *System {
	return &System{
		Particles:    make([]Particle, 0, maxParticles),
		MaxParticles: maxParticles,
	}
}

// AddEmitter adds an emitter to the system
func (s *System) AddEmitter(e *Emitter) {
	s.Emitters = append(s.Emitters, e)
}

// RemoveEmitter removes an emitter
func (s *System) RemoveEmitter(e *Emitter) {
	for i, em := range s.Emitters {
		if em == e {
			s.Emitters = append(s.Emitters[:i], s.Emitters[i+1:]...)
			return
		}
	}
}

// Update updates all particles and emitters
func (s *System) Update(dt float32) {
	// Update emitters and spawn particles
	for _, emitter := range s.Emitters {
		s.updateEmitter(emitter, dt)
	}

	// Update existing particles
	aliveCount := 0
	for i := range s.Particles {
		p := &s.Particles[i]
		if p.Life <= 0 {
			continue
		}

		// Update life
		p.Life -= dt / p.MaxLife
		if p.Life <= 0 {
			continue
		}

		// Apply physics
		p.Velocity.X += s.getGravity().X * dt
		p.Velocity.Y += s.getGravity().Y * dt
		p.Velocity.Z += s.getGravity().Z * dt

		// Apply drag
		drag := float32(1.0) - 0.1*dt
		p.Velocity.X *= drag
		p.Velocity.Y *= drag
		p.Velocity.Z *= drag

		// Update position
		p.Position.X += p.Velocity.X * dt
		p.Position.Y += p.Velocity.Y * dt
		p.Position.Z += p.Velocity.Z * dt

		// Update rotation
		p.Rotation += p.RotSpeed * dt

		// Keep alive particles at the front
		if aliveCount != i {
			s.Particles[aliveCount] = s.Particles[i]
		}
		aliveCount++
	}

	// Trim dead particles
	s.Particles = s.Particles[:aliveCount]
}

func (s *System) getGravity() emath.Vec3 {
	// Default gravity if no emitters
	return emath.Vec3{Y: -9.8}
}

func (s *System) updateEmitter(e *Emitter, dt float32) {
	// Handle burst
	if e.Burst > 0 {
		for i := 0; i < e.Burst && len(s.Particles) < s.MaxParticles; i++ {
			s.spawnParticle(e)
		}
		e.Burst = 0
	}

	// Handle continuous emission
	if e.Rate > 0 {
		e.accumulator += dt
		interval := 1.0 / e.Rate
		for e.accumulator >= interval && len(s.Particles) < s.MaxParticles {
			s.spawnParticle(e)
			e.accumulator -= interval
		}
	}

	// Handle burst interval
	if e.BurstInterval > 0 {
		e.burstTimer += dt
		if e.burstTimer >= e.BurstInterval {
			e.burstTimer = 0
			e.Burst = 10 // Default burst count
		}
	}
}

func (s *System) spawnParticle(e *Emitter) {
	// Calculate emission direction with spread
	dir := e.Direction
	if e.Spread > 0 {
		dir = randomConeDirection(e.Direction, e.Spread)
	}

	// Randomize properties
	speed := randomRange(e.InitialSpeed)
	size := randomRange(e.InitialSize)
	life := randomRange(e.Lifetime)
	rot := randomRange(e.InitialRotation)
	rotSpeed := randomRange(e.RotationSpeed)

	p := Particle{
		Position: e.Position,
		Velocity: emath.Vec3{
			X: dir.X * speed,
			Y: dir.Y * speed,
			Z: dir.Z * speed,
		},
		Color:    e.InitialColor,
		Size:     size,
		Life:     1.0,
		MaxLife:  life,
		Rotation: rot,
		RotSpeed: rotSpeed,
	}

	s.Particles = append(s.Particles, p)
}

// GetParticleColor returns interpolated color based on lifetime
func (e *Emitter) GetParticleColor(life float32) color.RGBA {
	t := 1.0 - life // 0 at start, 1 at end
	return color.RGBA{
		R: uint8(float32(e.InitialColor.R)*(1-t) + float32(e.EndColor.R)*t),
		G: uint8(float32(e.InitialColor.G)*(1-t) + float32(e.EndColor.G)*t),
		B: uint8(float32(e.InitialColor.B)*(1-t) + float32(e.EndColor.B)*t),
		A: uint8(float32(e.InitialColor.A)*(1-t) + float32(e.EndColor.A)*t),
	}
}

// GetParticleSize returns size based on lifetime curve
func (e *Emitter) GetParticleSize(baseSize, life float32) float32 {
	if len(e.SizeOverLife.Points) == 0 {
		return baseSize
	}
	return baseSize * e.SizeOverLife.Evaluate(1.0-life)
}

// Evaluate evaluates a curve at time t (0-1)
func (c *Curve) Evaluate(t float32) float32 {
	if len(c.Points) == 0 {
		return 1.0
	}
	if len(c.Points) == 1 {
		return c.Points[0].Value
	}

	// Find surrounding points
	for i := 0; i < len(c.Points)-1; i++ {
		if t >= c.Points[i].Time && t <= c.Points[i+1].Time {
			// Lerp between points
			localT := (t - c.Points[i].Time) / (c.Points[i+1].Time - c.Points[i].Time)
			return c.Points[i].Value + (c.Points[i+1].Value-c.Points[i].Value)*localT
		}
	}

	return c.Points[len(c.Points)-1].Value
}

// Helper functions
func randomRange(r Range) float32 {
	if r.Max <= r.Min {
		return r.Min
	}
	return r.Min + rand.Float32()*(r.Max-r.Min)
}

func randomConeDirection(baseDir emath.Vec3, spreadDegrees float32) emath.Vec3 {
	// Convert spread to radians
	spreadRad := spreadDegrees * math.Pi / 180.0

	// Random angle within spread cone
	theta := rand.Float64() * float64(spreadRad)
	phi := rand.Float64() * 2 * math.Pi

	// Calculate offset from base direction
	sinTheta := math.Sin(theta)
	cosTheta := math.Cos(theta)
	sinPhi := math.Sin(phi)
	cosPhi := math.Cos(phi)

	// Create a coordinate system around the base direction
	up := emath.Vec3{Y: 1}
	if math.Abs(float64(baseDir.Y)) > 0.99 {
		up = emath.Vec3{X: 1}
	}

	// Cross products to get perpendicular vectors
	right := normalize(cross(up, baseDir))
	newUp := cross(baseDir, right)

	// Calculate final direction
	return normalize(emath.Vec3{
		X: baseDir.X*float32(cosTheta) + right.X*float32(sinTheta*cosPhi) + newUp.X*float32(sinTheta*sinPhi),
		Y: baseDir.Y*float32(cosTheta) + right.Y*float32(sinTheta*cosPhi) + newUp.Y*float32(sinTheta*sinPhi),
		Z: baseDir.Z*float32(cosTheta) + right.Z*float32(sinTheta*cosPhi) + newUp.Z*float32(sinTheta*sinPhi),
	})
}

func normalize(v emath.Vec3) emath.Vec3 {
	len := float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
	if len < 0.0001 {
		return emath.Vec3{Y: 1}
	}
	return emath.Vec3{X: v.X / len, Y: v.Y / len, Z: v.Z / len}
}

func cross(a, b emath.Vec3) emath.Vec3 {
	return emath.Vec3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

// Preset emitters for common effects

// NewFireEmitter creates a fire effect emitter
func NewFireEmitter(pos emath.Vec3) *Emitter {
	return &Emitter{
		Position:     pos,
		Direction:    emath.Vec3{Y: 1},
		Spread:       30,
		Rate:         50,
		InitialSpeed: Range{Min: 2, Max: 4},
		InitialSize:  Range{Min: 0.3, Max: 0.6},
		Lifetime:     Range{Min: 0.5, Max: 1.5},
		InitialColor: color.RGBA{R: 255, G: 200, B: 50, A: 255},
		EndColor:     color.RGBA{R: 255, G: 50, B: 0, A: 0},
		Gravity:      emath.Vec3{Y: 2}, // Fire rises
		SizeOverLife: Curve{Points: []CurvePoint{
			{Time: 0, Value: 0.5},
			{Time: 0.3, Value: 1.0},
			{Time: 1, Value: 0},
		}},
	}
}

// NewSmokeEmitter creates a smoke effect emitter
func NewSmokeEmitter(pos emath.Vec3) *Emitter {
	return &Emitter{
		Position:     pos,
		Direction:    emath.Vec3{Y: 1},
		Spread:       45,
		Rate:         20,
		InitialSpeed: Range{Min: 0.5, Max: 1.5},
		InitialSize:  Range{Min: 0.5, Max: 1.0},
		Lifetime:     Range{Min: 2, Max: 4},
		InitialColor: color.RGBA{R: 100, G: 100, B: 100, A: 200},
		EndColor:     color.RGBA{R: 150, G: 150, B: 150, A: 0},
		Gravity:      emath.Vec3{Y: 0.5},
		SizeOverLife: Curve{Points: []CurvePoint{
			{Time: 0, Value: 0.5},
			{Time: 0.5, Value: 1.0},
			{Time: 1, Value: 1.5},
		}},
	}
}

// NewExplosionEmitter creates an explosion effect
func NewExplosionEmitter(pos emath.Vec3) *Emitter {
	return &Emitter{
		Position:     pos,
		Direction:    emath.Vec3{Y: 1},
		Spread:       180, // All directions
		Burst:        100,
		InitialSpeed: Range{Min: 5, Max: 15},
		InitialSize:  Range{Min: 0.2, Max: 0.5},
		Lifetime:     Range{Min: 0.3, Max: 0.8},
		InitialColor: color.RGBA{R: 255, G: 255, B: 200, A: 255},
		EndColor:     color.RGBA{R: 255, G: 100, B: 0, A: 0},
		Gravity:      emath.Vec3{Y: -5},
	}
}

// NewWaterSplashEmitter creates a water splash effect
func NewWaterSplashEmitter(pos emath.Vec3) *Emitter {
	return &Emitter{
		Position:     pos,
		Direction:    emath.Vec3{Y: 1},
		Spread:       60,
		Burst:        50,
		InitialSpeed: Range{Min: 3, Max: 8},
		InitialSize:  Range{Min: 0.1, Max: 0.3},
		Lifetime:     Range{Min: 0.5, Max: 1.0},
		InitialColor: color.RGBA{R: 150, G: 200, B: 255, A: 200},
		EndColor:     color.RGBA{R: 100, G: 150, B: 255, A: 0},
		Gravity:      emath.Vec3{Y: -15},
	}
}

// NewSparkEmitter creates a spark/electric effect
func NewSparkEmitter(pos emath.Vec3) *Emitter {
	return &Emitter{
		Position:      pos,
		Direction:     emath.Vec3{Y: 1},
		Spread:        90,
		Rate:          30,
		InitialSpeed:  Range{Min: 8, Max: 15},
		InitialSize:   Range{Min: 0.05, Max: 0.15},
		Lifetime:      Range{Min: 0.1, Max: 0.3},
		InitialColor:  color.RGBA{R: 255, G: 255, B: 255, A: 255},
		EndColor:      color.RGBA{R: 255, G: 200, B: 100, A: 0},
		Gravity:       emath.Vec3{Y: -20},
		RotationSpeed: Range{Min: -360, Max: 360},
	}
}
