package animation

import (
	"math"
	"testing"

	emath "goenginekenga/engine/math"
)

// testSkeleton — простой скелет: root + дочерняя кость arm.
func testSkeleton() *Skeleton {
	return &Skeleton{
		Bones: []Bone{
			{Name: "root", ParentIndex: -1, LocalBind: Transform{Scale: emath.V3(1, 1, 1)}, InvBindPose: identityMatrix()},
			{Name: "arm", ParentIndex: 0, LocalBind: Transform{Position: emath.V3(0, 1, 0), Scale: emath.V3(1, 1, 1)}, InvBindPose: identityMatrix()},
		},
	}
}

// slideClip — клип, двигающий root по X от 0 до 10 за 1 секунду.
func slideClip(axis string, loop bool) *Clip {
	c := NewClip("slide"+axis, 1.0)
	c.Loop = loop
	c.AddTrack("root", []Keyframe{
		{Time: 0, Position: emath.V3(0, 0, 0), Scale: emath.V3(1, 1, 1)},
		{Time: 1, Position: emath.V3(10, 0, 0), Scale: emath.V3(1, 1, 1)},
	})
	_ = axis
	return c
}

// boneX читает трансляцию X из bone-матрицы (column-major: [12]).
func boneX(m Matrix4x4) float32 { return m[12] }

func TestAnimatorPlayback(t *testing.T) {
	anim := NewAnimator(testSkeleton())
	clip := slideClip("x", false)
	anim.Play(clip)

	if !anim.Playing {
		t.Fatal("should be playing after Play")
	}
	anim.Update(0.5)
	if got := boneX(anim.GetBoneMatrix(0)); math.Abs(float64(got)-5) > 0.01 {
		t.Fatalf("bone X at t=0.5: %v, want 5", got)
	}
	// Дочерняя кость наследует позицию родителя (world matrix)
	if got := boneX(anim.GetBoneMatrix(1)); math.Abs(float64(got)-5) > 0.01 {
		t.Fatalf("child bone X at t=0.5: %v, want 5", got)
	}
}

func TestAnimatorLooping(t *testing.T) {
	anim := NewAnimator(testSkeleton())
	clip := slideClip("x", true)
	anim.Play(clip)

	// 1.3s при длительности 1s: время оборачивается на 0.3 → X = 3
	anim.Update(1.3)
	if !anim.Playing {
		t.Fatal("looping clip must keep playing")
	}
	if got := boneX(anim.GetBoneMatrix(0)); math.Abs(float64(got)-3) > 0.05 {
		t.Fatalf("bone X after loop: %v, want 3", got)
	}
}

func TestAnimatorNonLoopStops(t *testing.T) {
	anim := NewAnimator(testSkeleton())
	clip := slideClip("x", false)
	anim.Play(clip)

	anim.Update(2.0)
	if anim.Playing {
		t.Fatal("non-looping clip must stop after end")
	}
	if got := boneX(anim.GetBoneMatrix(0)); math.Abs(float64(got)-10) > 0.05 {
		t.Fatalf("bone X at end: %v, want 10 (clamped)", got)
	}
}

func TestAnimatorCrossfade(t *testing.T) {
	anim := NewAnimator(testSkeleton())
	a := slideClip("x", false)
	b := NewClip("rise", 1.0)
	b.AddTrack("root", []Keyframe{
		{Time: 0, Position: emath.V3(0, 0, 0), Scale: emath.V3(1, 1, 1)},
		{Time: 1, Position: emath.V3(0, 10, 0), Scale: emath.V3(1, 1, 1)},
	})

	anim.Play(a)
	anim.Update(0.5) // X = 5
	anim.PlayCrossfade(b, 1.0)

	// Ещё 0.5с: время клипа A дошло до конца (X=10), Next-клип сэмплится с t=0 (X=0),
	// blend=0.5 → X = lerp(10, 0, 0.5) = 5
	anim.Update(0.5)
	if anim.NextClip == nil {
		t.Fatal("crossfade should still be in progress")
	}
	if got := boneX(anim.GetBoneMatrix(0)); math.Abs(float64(got)-5) > 0.05 {
		t.Fatalf("blended X: %v, want 5 (lerp(10,0,0.5))", got)
	}

	// Кроссфейд завершён: играет clip b
	anim.Update(0.6)
	if anim.CurrentClip != b || anim.NextClip != nil {
		t.Fatalf("crossfade not completed: current=%p next=%p", anim.CurrentClip, anim.NextClip)
	}
}
