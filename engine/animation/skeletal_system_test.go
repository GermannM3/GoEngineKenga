package animation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/ecs"
)

// TestQuatToEuler проверяет, что quaternion -> euler даёт ожидаемые градусы.
func TestQuatToEuler(t *testing.T) {
	// Поворот на 90° вокруг Y: quat (cos45, 0, sin45, 0)
	r := quatToEulerDegrees([4]float32{0.70710678, 0, 0.70710678, 0})
	if r.Y < 89 || r.Y > 91 {
		t.Fatalf("expected Y≈90, got %v", r)
	}
	if r.X != 0 || r.Z != 0 {
		t.Fatalf("expected X=Z=0, got %v", r)
	}

	// Identity quaternion
	r = quatToEulerDegrees([4]float32{1, 0, 0, 0})
	if r.X != 0 || r.Y != 0 || r.Z != 0 {
		t.Fatalf("expected zero euler for identity, got %v", r)
	}
}

// TestAssetSkeletonToSkeleton проверяет конвертацию скелета из ассета.
func TestAssetSkeletonToSkeleton(t *testing.T) {
	a := &asset.Skeleton{
		JointNames:    []string{"root", "leg"},
		ParentIndices: []int{-1, 0},
		InverseBindMatrices: [][16]float32{
			{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1},
			{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 1, 0, 1},
		},
	}
	sk := AssetSkeletonToSkeleton(a)
	if len(sk.Bones) != 2 {
		t.Fatalf("expected 2 bones, got %d", len(sk.Bones))
	}
	if sk.Bones[0].ParentIndex != -1 || sk.Bones[1].ParentIndex != 0 {
		t.Fatalf("parent indices not preserved: %d, %d", sk.Bones[0].ParentIndex, sk.Bones[1].ParentIndex)
	}
	if sk.Bones[1].InvBindPose[15] != 1 {
		t.Fatalf("inverse bind matrix not copied")
	}
}

// TestAssetClipToClip проверяет конвертацию клипа (quaternion -> euler degrees).
func TestAssetClipToClip(t *testing.T) {
	a := &asset.AnimationClip{
		Name:     "walk",
		Duration: 1.0,
		Loop:     true,
		Tracks: []asset.AnimationTrack{
			{
				NodeName: "root",
				Keyframes: []asset.Keyframe{
					{Time: 0, Position: [3]float32{0, 0, 0}, Rotation: [4]float32{1, 0, 0, 0}, Scale: [3]float32{1, 1, 1}},
					{Time: 0.5, Position: [3]float32{0, 0.5, 0}, Rotation: [4]float32{0.70710678, 0, 0.70710678, 0}, Scale: [3]float32{1, 1, 1}},
				},
			},
		},
	}
	c := AssetClipToClip(a)
	if c == nil || c.Name != "walk" || !c.Loop || c.Duration != 1.0 {
		t.Fatalf("clip metadata lost: %+v", c)
	}
	if len(c.Tracks) != 1 || len(c.Tracks[0].Keyframes) != 2 {
		t.Fatalf("tracks/keyframes lost")
	}
	// Y-поворот 90° в кадре 2
	if c.Tracks[0].Keyframes[1].Rotation.Y < 89 || c.Tracks[0].Keyframes[1].Rotation.Y > 91 {
		t.Fatalf("expected euler Y≈90, got %v", c.Tracks[0].Keyframes[1].Rotation)
	}
}

// writeTestAssets создаёт skeleton.json и clip.json во временной директории.
func writeTestAssets(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	sk := asset.Skeleton{
		Name:          "test",
		JointNames:    []string{"root", "child"},
		ParentIndices: []int{-1, 0},
		InverseBindMatrices: [][16]float32{
			{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1},
			{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 1, 0, 1},
		},
	}
	writeJSON(t, filepath.Join(dir, "skeleton.json"), sk)

	clip := asset.AnimationClip{
		Name:     "bob",
		Duration: 1.0,
		Loop:     true,
		Tracks: []asset.AnimationTrack{
			{
				NodeName: "root",
				Keyframes: []asset.Keyframe{
					{Time: 0, Position: [3]float32{0, 0, 0}, Rotation: [4]float32{1, 0, 0, 0}, Scale: [3]float32{1, 1, 1}},
					{Time: 0.5, Position: [3]float32{0, 1, 0}, Rotation: [4]float32{1, 0, 0, 0}, Scale: [3]float32{1, 1, 1}},
					{Time: 1.0, Position: [3]float32{0, 0, 0}, Rotation: [4]float32{1, 0, 0, 0}, Scale: [3]float32{1, 1, 1}},
				},
			},
		},
	}
	writeJSON(t, filepath.Join(dir, "clip.json"), clip)
	return dir
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestSkeletalAnimationSystem проверяет, что система загружает скелет/клип из ассетов
// и пишет меняющиеся во времени bone matrices в ecs.Animator.
func TestSkeletalAnimationSystem(t *testing.T) {
	dir := writeTestAssets(t)

	w := ecs.NewWorld()
	id := w.CreateEntity("Animated")
	w.SetAnimator(id, ecs.Animator{
		SkeletonPath: "skeleton.json",
		ClipPaths:    []string{"clip.json"},
		Clip:         "bob",
		Speed:        1,
		Loop:         true,
		AutoPlay:     true,
	})

	sys := NewSkeletalAnimationSystem(dir)

	// t=0: анимация должна стартовать
	sys.Update(w, 0.0)
	a1, ok := w.GetAnimator(id)
	if !ok {
		t.Fatal("animator component lost")
	}
	// 2 кости x 16 floats
	if len(a1.BoneMatrices) != 32 {
		t.Fatalf("expected 32 floats (2 bones), got %d", len(a1.BoneMatrices))
	}
	first := append([]float32(nil), a1.BoneMatrices...)

	// t=0.25: матрицы должны измениться (позиция root = 0.5 по Y)
	sys.Update(w, 0.25)
	a2, _ := w.GetAnimator(id)
	if len(a2.BoneMatrices) != 32 {
		t.Fatalf("bone matrices lost after update: %d", len(a2.BoneMatrices))
	}
	// Root-кость: translation Y (column-major: элемент 13) ≈ 0.5
	if a2.BoneMatrices[13] < 0.4 || a2.BoneMatrices[13] > 0.6 {
		t.Fatalf("expected root Y translation ≈ 0.5, got %v", a2.BoneMatrices[13])
	}
	changed := false
	for i := range first {
		if first[i] != a2.BoneMatrices[i] {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("bone matrices did not change over time")
	}

	// Переключение клипа не должно ломать систему
	a3 := a2
	a3.Clip = "bob"
	w.SetAnimator(id, a3)
	sys.Update(w, 0.25)
	if _, ok := w.GetAnimator(id); !ok {
		t.Fatal("animator lost after clip switch")
	}
}
