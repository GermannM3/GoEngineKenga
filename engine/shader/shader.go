package shader

import (
	"image/color"
	"math"

	emath "goenginekenga/engine/math"
)

// Shader represents a programmable shader
type Shader struct {
	Name         string
	VertexFunc   VertexShaderFunc
	FragmentFunc FragmentShaderFunc
	Uniforms     map[string]interface{}
}

// VertexInput is input to vertex shader
type VertexInput struct {
	Position emath.Vec3
	Normal   emath.Vec3
	UV       emath.Vec2
	Color    color.RGBA
}

// VertexOutput is output from vertex shader
type VertexOutput struct {
	Position      emath.Vec3 // Clip space
	WorldPosition emath.Vec3
	WorldNormal   emath.Vec3
	UV            emath.Vec2
	Color         color.RGBA
	Custom        map[string]float32 // Custom interpolated values
}

// FragmentInput is input to fragment shader
type FragmentInput struct {
	Position      emath.Vec2 // Screen space
	WorldPosition emath.Vec3
	WorldNormal   emath.Vec3
	UV            emath.Vec2
	Color         color.RGBA
	Depth         float32
	Custom        map[string]float32
}

// VertexShaderFunc is the vertex shader function signature
type VertexShaderFunc func(input VertexInput, uniforms map[string]interface{}) VertexOutput

// FragmentShaderFunc is the fragment shader function signature
type FragmentShaderFunc func(input FragmentInput, uniforms map[string]interface{}) color.RGBA

// NewShader creates a new shader
func NewShader(name string) *Shader {
	return &Shader{
		Name:     name,
		Uniforms: make(map[string]interface{}),
	}
}

// SetUniform sets a uniform value
func (s *Shader) SetUniform(name string, value interface{}) {
	s.Uniforms[name] = value
}

// GetFloat gets a float uniform
func GetFloat(uniforms map[string]interface{}, name string, defaultVal float32) float32 {
	if v, ok := uniforms[name].(float32); ok {
		return v
	}
	if v, ok := uniforms[name].(float64); ok {
		return float32(v)
	}
	return defaultVal
}

// GetVec3 gets a Vec3 uniform
func GetVec3(uniforms map[string]interface{}, name string, defaultVal emath.Vec3) emath.Vec3 {
	if v, ok := uniforms[name].(emath.Vec3); ok {
		return v
	}
	return defaultVal
}

// GetColor gets a color uniform
func GetColor(uniforms map[string]interface{}, name string, defaultVal color.RGBA) color.RGBA {
	if v, ok := uniforms[name].(color.RGBA); ok {
		return v
	}
	return defaultVal
}

// Built-in shaders

// DefaultVertexShader is a basic vertex shader
func DefaultVertexShader(input VertexInput, uniforms map[string]interface{}) VertexOutput {
	return VertexOutput{
		Position:      input.Position,
		WorldPosition: input.Position,
		WorldNormal:   input.Normal,
		UV:            input.UV,
		Color:         input.Color,
	}
}

// DefaultFragmentShader is a basic fragment shader
func DefaultFragmentShader(input FragmentInput, uniforms map[string]interface{}) color.RGBA {
	return input.Color
}

// UnlitFragmentShader renders without lighting
func UnlitFragmentShader(input FragmentInput, uniforms map[string]interface{}) color.RGBA {
	baseColor := GetColor(uniforms, "color", input.Color)
	return baseColor
}

// ToonFragmentShader creates a cel-shaded look
func ToonFragmentShader(input FragmentInput, uniforms map[string]interface{}) color.RGBA {
	baseColor := GetColor(uniforms, "color", input.Color)
	lightDir := GetVec3(uniforms, "lightDir", emath.Vec3{X: 0.5, Y: 1, Z: 0.5})

	// Normalize
	len := float32(math.Sqrt(float64(lightDir.X*lightDir.X + lightDir.Y*lightDir.Y + lightDir.Z*lightDir.Z)))
	if len > 0 {
		lightDir.X /= len
		lightDir.Y /= len
		lightDir.Z /= len
	}

	// Calculate dot product
	dot := input.WorldNormal.X*lightDir.X + input.WorldNormal.Y*lightDir.Y + input.WorldNormal.Z*lightDir.Z
	if dot < 0 {
		dot = 0
	}

	// Quantize to create toon bands
	levels := GetFloat(uniforms, "levels", 3)
	dot = float32(math.Floor(float64(dot*levels))) / levels

	// Apply lighting
	return color.RGBA{
		R: uint8(float32(baseColor.R) * (0.3 + dot*0.7)),
		G: uint8(float32(baseColor.G) * (0.3 + dot*0.7)),
		B: uint8(float32(baseColor.B) * (0.3 + dot*0.7)),
		A: baseColor.A,
	}
}

// WaveDistortShader creates a wavy distortion effect
func WaveDistortShader(input FragmentInput, uniforms map[string]interface{}) color.RGBA {
	baseColor := input.Color
	time := GetFloat(uniforms, "time", 0)
	amplitude := GetFloat(uniforms, "amplitude", 0.1)
	frequency := GetFloat(uniforms, "frequency", 10)

	// Offset UV based on sine wave
	offsetX := amplitude * float32(math.Sin(float64(input.UV.Y*frequency+time)))
	offsetY := amplitude * float32(math.Cos(float64(input.UV.X*frequency+time)))

	_ = offsetX
	_ = offsetY
	// In a real implementation, you'd sample a texture with the offset UVs

	// For now, just tint based on the wave
	waveFactor := float32(math.Sin(float64(input.UV.X*frequency+input.UV.Y*frequency+time)))*0.5 + 0.5

	return color.RGBA{
		R: uint8(float32(baseColor.R) * (0.8 + waveFactor*0.2)),
		G: uint8(float32(baseColor.G) * (0.8 + waveFactor*0.2)),
		B: uint8(float32(baseColor.B) * (0.8 + waveFactor*0.2)),
		A: baseColor.A,
	}
}

// PsychedelicShader creates trippy color effects
func PsychedelicShader(input FragmentInput, uniforms map[string]interface{}) color.RGBA {
	time := GetFloat(uniforms, "time", 0)
	intensity := GetFloat(uniforms, "intensity", 1)

	// Create swirling color patterns
	x := input.UV.X - 0.5
	y := input.UV.Y - 0.5
	dist := float32(math.Sqrt(float64(x*x + y*y)))
	angle := float32(math.Atan2(float64(y), float64(x)))

	// Spiral pattern
	spiral := angle*2 + dist*10 - time*2

	r := float32(math.Sin(float64(spiral)))*0.5 + 0.5
	g := float32(math.Sin(float64(spiral+2.094)))*0.5 + 0.5
	b := float32(math.Sin(float64(spiral+4.188)))*0.5 + 0.5

	// Mix with intensity
	r = r*intensity + float32(input.Color.R)/255*(1-intensity)
	g = g*intensity + float32(input.Color.G)/255*(1-intensity)
	b = b*intensity + float32(input.Color.B)/255*(1-intensity)

	return color.RGBA{
		R: uint8(clamp01(r) * 255),
		G: uint8(clamp01(g) * 255),
		B: uint8(clamp01(b) * 255),
		A: input.Color.A,
	}
}

// GlitchShader creates a digital glitch effect
func GlitchShader(input FragmentInput, uniforms map[string]interface{}) color.RGBA {
	baseColor := input.Color
	time := GetFloat(uniforms, "time", 0)
	glitchAmount := GetFloat(uniforms, "glitch", 0.5)

	// Random glitch offset
	seed := float64(int(input.Position.Y)%10) + float64(time*1000)
	glitchOffset := float32(math.Sin(seed) * float64(glitchAmount))

	// Color channel separation
	if glitchOffset > 0.3 {
		// Red shift
		return color.RGBA{
			R: uint8(min(255, int(baseColor.R)+int(glitchAmount*50))),
			G: baseColor.G,
			B: uint8(max(0, int(baseColor.B)-int(glitchAmount*50))),
			A: baseColor.A,
		}
	}

	return baseColor
}

// FogShader applies distance fog
func FogShader(input FragmentInput, uniforms map[string]interface{}) color.RGBA {
	baseColor := input.Color
	fogColor := GetColor(uniforms, "fogColor", color.RGBA{R: 200, G: 200, B: 220, A: 255})
	fogDensity := GetFloat(uniforms, "fogDensity", 0.05)
	fogStart := GetFloat(uniforms, "fogStart", 10)

	// Calculate fog based on depth
	fogFactor := 1.0 - float32(math.Exp(float64(-fogDensity*(input.Depth-fogStart))))
	if fogFactor < 0 {
		fogFactor = 0
	}
	if fogFactor > 1 {
		fogFactor = 1
	}

	// Blend with fog color
	return color.RGBA{
		R: uint8(float32(baseColor.R)*(1-fogFactor) + float32(fogColor.R)*fogFactor),
		G: uint8(float32(baseColor.G)*(1-fogFactor) + float32(fogColor.G)*fogFactor),
		B: uint8(float32(baseColor.B)*(1-fogFactor) + float32(fogColor.B)*fogFactor),
		A: baseColor.A,
	}
}

// OutlineShader adds an outline effect (post-process)
type OutlineShader struct {
	OutlineColor  color.RGBA
	Thickness     float32
	DepthBuffer   []float32
	Width, Height int
}

func (s *OutlineShader) Process(x, y int, baseColor color.RGBA) color.RGBA {
	if s.DepthBuffer == nil {
		return baseColor
	}

	// Check depth difference with neighbors
	centerDepth := s.getDepth(x, y)
	maxDiff := float32(0)

	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			neighborDepth := s.getDepth(x+dx, y+dy)
			diff := float32(math.Abs(float64(neighborDepth - centerDepth)))
			if diff > maxDiff {
				maxDiff = diff
			}
		}
	}

	// If depth difference is significant, draw outline
	if maxDiff > s.Thickness {
		return s.OutlineColor
	}

	return baseColor
}

func (s *OutlineShader) getDepth(x, y int) float32 {
	if x < 0 || x >= s.Width || y < 0 || y >= s.Height {
		return 1.0
	}
	return s.DepthBuffer[y*s.Width+x]
}

// PostProcessChain chains multiple post-process effects
type PostProcessChain struct {
	Effects []PostProcessEffect
}

// PostProcessEffect is a post-process effect interface
type PostProcessEffect interface {
	Process(x, y int, c color.RGBA) color.RGBA
}

// Apply applies all effects in the chain
func (chain *PostProcessChain) Apply(x, y int, c color.RGBA) color.RGBA {
	result := c
	for _, effect := range chain.Effects {
		result = effect.Process(x, y, result)
	}
	return result
}

// VignetteEffect darkens the edges of the screen
type VignetteEffect struct {
	Intensity     float32
	Width, Height int
}

func (v *VignetteEffect) Process(x, y int, c color.RGBA) color.RGBA {
	// Calculate distance from center
	cx := float32(v.Width) / 2
	cy := float32(v.Height) / 2
	dx := float32(x) - cx
	dy := float32(y) - cy
	maxDist := float32(math.Sqrt(float64(cx*cx + cy*cy)))
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))

	// Vignette factor
	vignette := 1 - (dist/maxDist)*v.Intensity
	if vignette < 0 {
		vignette = 0
	}

	return color.RGBA{
		R: uint8(float32(c.R) * vignette),
		G: uint8(float32(c.G) * vignette),
		B: uint8(float32(c.B) * vignette),
		A: c.A,
	}
}

// ChromaticAberrationEffect separates color channels
type ChromaticAberrationEffect struct {
	Intensity     float32
	Width, Height int
	SourceBuffer  []color.RGBA
}

func (ca *ChromaticAberrationEffect) Process(x, y int, c color.RGBA) color.RGBA {
	if ca.SourceBuffer == nil {
		return c
	}

	offset := int(ca.Intensity * float32(ca.Width) * 0.01)

	// Sample red from left, blue from right
	redX := x - offset
	blueX := x + offset

	r := c.R
	b := c.B

	if redX >= 0 && redX < ca.Width {
		r = ca.SourceBuffer[y*ca.Width+redX].R
	}
	if blueX >= 0 && blueX < ca.Width {
		b = ca.SourceBuffer[y*ca.Width+blueX].B
	}

	return color.RGBA{R: r, G: c.G, B: b, A: c.A}
}

// Helper functions
func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
