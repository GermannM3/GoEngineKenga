package animation

import (
	"math"

	"goenginekenga/engine/asset"
	emath "goenginekenga/engine/math"
)

// boneMatrixCount — максимум костей, загружаемых в GPU uniform (совпадает с WebGPU shader).
const boneMatrixCount = 64

// AssetSkeletonToSkeleton конвертирует asset.Skeleton (из glTF Skin) в animation.Skeleton.
// LocalBind = identity: финальная матрица кости = world(анимированной иерархии) * InvBindPose,
// что соответствует стандартной формуле скиннинга pos' = Σ w * (world_i * invBind_i * pos).
func AssetSkeletonToSkeleton(s *asset.Skeleton) *Skeleton {
	if s == nil {
		return nil
	}
	sk := &Skeleton{Bones: make([]Bone, len(s.JointNames))}
	for i, name := range s.JointNames {
		parent := -1
		if i < len(s.ParentIndices) {
			parent = s.ParentIndices[i]
		}
		sk.Bones[i] = Bone{
			Name:        name,
			ParentIndex: parent,
			LocalBind:   Transform{Scale: emath.Vec3{X: 1, Y: 1, Z: 1}},
			InvBindPose: Matrix4x4{},
		}
		if i < len(s.InverseBindMatrices) {
			sk.Bones[i].InvBindPose = Matrix4x4(s.InverseBindMatrices[i])
		}
	}
	return sk
}

// AssetClipToClip конвертирует asset.AnimationClip (quaternion-кадры из glTF) в animation.Clip
// (euler degrees, как использует Animator).
func AssetClipToClip(a *asset.AnimationClip) *Clip {
	if a == nil {
		return nil
	}
	clip := &Clip{
		Name:     a.Name,
		Duration: a.Duration,
		Loop:     a.Loop,
		Speed:    1.0,
	}
	for _, t := range a.Tracks {
		kfs := make([]Keyframe, len(t.Keyframes))
		for i, k := range t.Keyframes {
			kfs[i] = Keyframe{
				Time:     k.Time,
				Position: emath.Vec3{X: k.Position[0], Y: k.Position[1], Z: k.Position[2]},
				Rotation: quatToEulerDegrees(k.Rotation),
				Scale:    emath.Vec3{X: k.Scale[0], Y: k.Scale[1], Z: k.Scale[2]},
			}
		}
		clip.AddTrack(t.NodeName, kfs)
	}
	return clip
}

// quatToEulerDegrees конвертирует quaternion (wxyz, как в glTF) в euler degrees (ZYX order —
// тот же, что использует transformToMatrix: R = Rz*Ry*Rx).
func quatToEulerDegrees(q [4]float32) emath.Vec3 {
	w, x, y, z := q[0], q[1], q[2], q[3]

	// X (roll)
	roll := math.Atan2(float64(2*(w*x+y*z)), float64(1-2*(x*x+y*y)))
	// Y (pitch)
	sinp := 2 * (w*y - z*x)
	if sinp > 1 {
		sinp = 1
	}
	if sinp < -1 {
		sinp = -1
	}
	pitch := math.Asin(float64(sinp))
	// Z (yaw)
	yaw := math.Atan2(float64(2*(w*z+x*y)), float64(1-2*(y*y+z*z)))

	return emath.Vec3{
		X: float32(roll * 180 / math.Pi),
		Y: float32(pitch * 180 / math.Pi),
		Z: float32(yaw * 180 / math.Pi),
	}
}

// ClampBoneCount ограничивает число костей до лимита GPU uniform.
func ClampBoneCount(n int) int {
	if n > boneMatrixCount {
		return boneMatrixCount
	}
	return n
}

// MaxBoneCount возвращает лимит костей, поддерживаемый GPU-скиннингом.
func MaxBoneCount() int { return boneMatrixCount }
