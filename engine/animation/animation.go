package animation

import (
	"math"

	emath "goenginekenga/engine/math"
)

// Keyframe represents a single keyframe in an animation
type Keyframe struct {
	Time     float32
	Position emath.Vec3
	Rotation emath.Vec3 // Euler angles in degrees
	Scale    emath.Vec3
}

// Track represents a single animation track (bone/transform)
type Track struct {
	Name      string
	Keyframes []Keyframe
}

// Clip represents a complete animation clip
type Clip struct {
	Name     string
	Duration float32
	Tracks   []Track
	Loop     bool
	Speed    float32
}

// Skeleton represents a bone hierarchy for skeletal animation
type Skeleton struct {
	Bones []Bone
}

// Bone represents a single bone in a skeleton
type Bone struct {
	Name        string
	ParentIndex int // -1 for root
	LocalBind   Transform
	InvBindPose Matrix4x4
}

// Transform represents position, rotation, scale
type Transform struct {
	Position emath.Vec3
	Rotation emath.Vec3
	Scale    emath.Vec3
}

// Matrix4x4 for bone transforms
type Matrix4x4 [16]float32

// Animator manages animation playback
type Animator struct {
	Skeleton      *Skeleton
	CurrentClip   *Clip
	NextClip      *Clip
	Time          float32
	BlendTime     float32
	BlendDuration float32
	Playing       bool

	// Computed bone transforms
	BoneTransforms []Transform
	WorldMatrices  []Matrix4x4
}

// NewAnimator creates a new animator
func NewAnimator(skeleton *Skeleton) *Animator {
	boneCount := 0
	if skeleton != nil {
		boneCount = len(skeleton.Bones)
	}
	return &Animator{
		Skeleton:       skeleton,
		BoneTransforms: make([]Transform, boneCount),
		WorldMatrices:  make([]Matrix4x4, boneCount),
		Playing:        false,
	}
}

// Play starts playing a clip
func (a *Animator) Play(clip *Clip) {
	a.CurrentClip = clip
	a.Time = 0
	a.Playing = true
}

// PlayCrossfade crossfades to a new clip
func (a *Animator) PlayCrossfade(clip *Clip, duration float32) {
	if a.CurrentClip == nil {
		a.Play(clip)
		return
	}
	a.NextClip = clip
	a.BlendTime = 0
	a.BlendDuration = duration
}

// Stop stops playback
func (a *Animator) Stop() {
	a.Playing = false
}

// Update updates the animation
func (a *Animator) Update(dt float32) {
	if !a.Playing || a.CurrentClip == nil {
		return
	}

	speed := a.CurrentClip.Speed
	if speed == 0 {
		speed = 1
	}

	// Update time
	a.Time += dt * speed

	// Handle looping
	if a.Time >= a.CurrentClip.Duration {
		if a.CurrentClip.Loop {
			a.Time = float32(math.Mod(float64(a.Time), float64(a.CurrentClip.Duration)))
		} else {
			a.Time = a.CurrentClip.Duration
			a.Playing = false
		}
	}

	// Handle crossfade
	if a.NextClip != nil {
		a.BlendTime += dt
		if a.BlendTime >= a.BlendDuration {
			a.CurrentClip = a.NextClip
			a.NextClip = nil
			a.Time = 0
			a.BlendTime = 0
		}
	}

	// Sample animation
	a.sampleAnimation()
}

func (a *Animator) sampleAnimation() {
	if a.Skeleton == nil || a.CurrentClip == nil {
		return
	}

	// Sample current clip
	currentTransforms := a.sampleClip(a.CurrentClip, a.Time)

	// Blend with next clip if crossfading
	if a.NextClip != nil && a.BlendDuration > 0 {
		nextTransforms := a.sampleClip(a.NextClip, 0)
		blend := a.BlendTime / a.BlendDuration
		for i := range currentTransforms {
			currentTransforms[i] = lerpTransform(currentTransforms[i], nextTransforms[i], blend)
		}
	}

	// Copy to bone transforms
	copy(a.BoneTransforms, currentTransforms)

	// Compute world matrices
	a.computeWorldMatrices()
}

func (a *Animator) sampleClip(clip *Clip, time float32) []Transform {
	transforms := make([]Transform, len(a.Skeleton.Bones))

	// Initialize with bind pose
	for i, bone := range a.Skeleton.Bones {
		transforms[i] = bone.LocalBind
	}

	// Apply track data
	for _, track := range clip.Tracks {
		boneIdx := a.findBoneIndex(track.Name)
		if boneIdx < 0 {
			continue
		}

		kf := sampleTrack(track, time)
		transforms[boneIdx] = kf
	}

	return transforms
}

func (a *Animator) findBoneIndex(name string) int {
	for i, bone := range a.Skeleton.Bones {
		if bone.Name == name {
			return i
		}
	}
	return -1
}

func sampleTrack(track Track, time float32) Transform {
	if len(track.Keyframes) == 0 {
		return Transform{Scale: emath.Vec3{X: 1, Y: 1, Z: 1}}
	}

	if len(track.Keyframes) == 1 {
		kf := track.Keyframes[0]
		return Transform{
			Position: kf.Position,
			Rotation: kf.Rotation,
			Scale:    kf.Scale,
		}
	}

	// Find surrounding keyframes
	var prev, next *Keyframe
	for i := 0; i < len(track.Keyframes)-1; i++ {
		if time >= track.Keyframes[i].Time && time <= track.Keyframes[i+1].Time {
			prev = &track.Keyframes[i]
			next = &track.Keyframes[i+1]
			break
		}
	}

	if prev == nil {
		kf := track.Keyframes[0]
		return Transform{Position: kf.Position, Rotation: kf.Rotation, Scale: kf.Scale}
	}

	// Interpolate
	t := (time - prev.Time) / (next.Time - prev.Time)
	return Transform{
		Position: lerpVec3(prev.Position, next.Position, t),
		Rotation: lerpVec3(prev.Rotation, next.Rotation, t), // Simple euler lerp
		Scale:    lerpVec3(prev.Scale, next.Scale, t),
	}
}

func (a *Animator) computeWorldMatrices() {
	for i, bone := range a.Skeleton.Bones {
		local := transformToMatrix(a.BoneTransforms[i])

		if bone.ParentIndex < 0 {
			a.WorldMatrices[i] = local
		} else {
			a.WorldMatrices[i] = multiplyMatrix(a.WorldMatrices[bone.ParentIndex], local)
		}
	}
}

// GetBoneMatrix returns the final bone matrix for skinning
func (a *Animator) GetBoneMatrix(boneIndex int) Matrix4x4 {
	if boneIndex < 0 || boneIndex >= len(a.WorldMatrices) {
		return identityMatrix()
	}
	return multiplyMatrix(a.WorldMatrices[boneIndex], a.Skeleton.Bones[boneIndex].InvBindPose)
}

// Helper functions
func lerpVec3(a, b emath.Vec3, t float32) emath.Vec3 {
	return emath.Vec3{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
		Z: a.Z + (b.Z-a.Z)*t,
	}
}

func lerpTransform(a, b Transform, t float32) Transform {
	return Transform{
		Position: lerpVec3(a.Position, b.Position, t),
		Rotation: lerpVec3(a.Rotation, b.Rotation, t),
		Scale:    lerpVec3(a.Scale, b.Scale, t),
	}
}

func transformToMatrix(t Transform) Matrix4x4 {
	// Create TRS matrix
	cos := func(deg float32) float32 { return float32(math.Cos(float64(deg * math.Pi / 180))) }
	sin := func(deg float32) float32 { return float32(math.Sin(float64(deg * math.Pi / 180))) }

	cx, sx := cos(t.Rotation.X), sin(t.Rotation.X)
	cy, sy := cos(t.Rotation.Y), sin(t.Rotation.Y)
	cz, sz := cos(t.Rotation.Z), sin(t.Rotation.Z)

	// Combined rotation matrix (ZYX order)
	return Matrix4x4{
		t.Scale.X * (cy * cz), t.Scale.X * (cy * sz), t.Scale.X * (-sy), 0,
		t.Scale.Y * (sx*sy*cz - cx*sz), t.Scale.Y * (sx*sy*sz + cx*cz), t.Scale.Y * (sx * cy), 0,
		t.Scale.Z * (cx*sy*cz + sx*sz), t.Scale.Z * (cx*sy*sz - sx*cz), t.Scale.Z * (cx * cy), 0,
		t.Position.X, t.Position.Y, t.Position.Z, 1,
	}
}

func multiplyMatrix(a, b Matrix4x4) Matrix4x4 {
	var result Matrix4x4
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			sum := float32(0)
			for k := 0; k < 4; k++ {
				sum += a[i+k*4] * b[k+j*4]
			}
			result[i+j*4] = sum
		}
	}
	return result
}

func identityMatrix() Matrix4x4 {
	return Matrix4x4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// NewClip creates a new animation clip
func NewClip(name string, duration float32) *Clip {
	return &Clip{
		Name:     name,
		Duration: duration,
		Speed:    1.0,
		Loop:     true,
	}
}

// AddTrack adds a track to a clip
func (c *Clip) AddTrack(name string, keyframes []Keyframe) {
	c.Tracks = append(c.Tracks, Track{Name: name, Keyframes: keyframes})
}

// CreateIdleAnimation creates a simple idle animation (breathing motion)
func CreateIdleAnimation() *Clip {
	clip := NewClip("idle", 2.0)
	clip.AddTrack("root", []Keyframe{
		{Time: 0.0, Position: emath.Vec3{Y: 0}, Scale: emath.Vec3{X: 1, Y: 1, Z: 1}},
		{Time: 1.0, Position: emath.Vec3{Y: 0.05}, Scale: emath.Vec3{X: 1, Y: 1.02, Z: 1}},
		{Time: 2.0, Position: emath.Vec3{Y: 0}, Scale: emath.Vec3{X: 1, Y: 1, Z: 1}},
	})
	return clip
}

// CreateWalkAnimation creates a simple walk cycle
func CreateWalkAnimation() *Clip {
	clip := NewClip("walk", 1.0)
	clip.AddTrack("root", []Keyframe{
		{Time: 0.0, Position: emath.Vec3{Y: 0}, Rotation: emath.Vec3{}},
		{Time: 0.25, Position: emath.Vec3{Y: 0.1}, Rotation: emath.Vec3{}},
		{Time: 0.5, Position: emath.Vec3{Y: 0}, Rotation: emath.Vec3{}},
		{Time: 0.75, Position: emath.Vec3{Y: 0.1}, Rotation: emath.Vec3{}},
		{Time: 1.0, Position: emath.Vec3{Y: 0}, Rotation: emath.Vec3{}},
	})
	return clip
}

// SpriteAnimation for 2D sprite animations

// SpriteFrame represents a single frame in a sprite animation
type SpriteFrame struct {
	X, Y, W, H int     // Source rectangle in spritesheet
	Duration   float32 // Duration of this frame in seconds
}

// SpriteClip represents a sprite animation clip
type SpriteClip struct {
	Name   string
	Frames []SpriteFrame
	Loop   bool
}

// SpriteAnimator manages sprite animation playback
type SpriteAnimator struct {
	CurrentClip *SpriteClip
	FrameIndex  int
	Time        float32
	Playing     bool
}

// NewSpriteAnimator creates a new sprite animator
func NewSpriteAnimator() *SpriteAnimator {
	return &SpriteAnimator{}
}

// Play starts a sprite animation
func (sa *SpriteAnimator) Play(clip *SpriteClip) {
	sa.CurrentClip = clip
	sa.FrameIndex = 0
	sa.Time = 0
	sa.Playing = true
}

// Update updates the sprite animation
func (sa *SpriteAnimator) Update(dt float32) {
	if !sa.Playing || sa.CurrentClip == nil || len(sa.CurrentClip.Frames) == 0 {
		return
	}

	sa.Time += dt

	// Check if we need to advance frame
	currentFrame := sa.CurrentClip.Frames[sa.FrameIndex]
	if sa.Time >= currentFrame.Duration {
		sa.Time -= currentFrame.Duration
		sa.FrameIndex++

		if sa.FrameIndex >= len(sa.CurrentClip.Frames) {
			if sa.CurrentClip.Loop {
				sa.FrameIndex = 0
			} else {
				sa.FrameIndex = len(sa.CurrentClip.Frames) - 1
				sa.Playing = false
			}
		}
	}
}

// GetCurrentFrame returns the current frame
func (sa *SpriteAnimator) GetCurrentFrame() *SpriteFrame {
	if sa.CurrentClip == nil || len(sa.CurrentClip.Frames) == 0 {
		return nil
	}
	return &sa.CurrentClip.Frames[sa.FrameIndex]
}
