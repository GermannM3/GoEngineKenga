package runtime

import (
	"math/rand"

	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
)

// CameraShake — тряска следящих камер (отдача удара, урон).
// Add включает эффект, Step вызывается каждый кадр после UpdateFollowCameras.
type CameraShake struct {
	timeLeft float32
	duration float32
	amp      float32
}

// Add запускает/усиливает тряску.
func (s *CameraShake) Add(duration, amplitude float32) {
	if duration > s.timeLeft {
		s.timeLeft = duration
		s.duration = duration
	}
	if amplitude > s.amp {
		s.amp = amplitude
	}
}

// Step затухает со временем и смещает все следящие камеры случайным образом.
func (s *CameraShake) Step(w *ecs.World, dt float32) {
	if s.timeLeft <= 0 {
		return
	}
	s.timeLeft -= dt
	k := s.timeLeft / s.duration // 1 → 0
	if k < 0 {
		k = 0
	}
	off := emath.Vec3{
		X: (rand.Float32()*2 - 1) * s.amp * k,
		Y: (rand.Float32()*2 - 1) * s.amp * 0.5 * k,
		Z: (rand.Float32()*2 - 1) * s.amp * k,
	}
	for _, id := range w.Entities() {
		cam, ok := w.GetCamera(id)
		if !ok || cam.FollowID == 0 {
			continue
		}
		tr, hasTr := w.GetTransform(id)
		if !hasTr {
			continue
		}
		tr.Position = tr.Position.Add(off)
		w.SetTransform(id, tr)
	}
	if s.timeLeft <= 0 {
		s.amp = 0
	}
}

// UpdateFollowCameras двигает 3D-камеры с FollowID за целью (плавно, если задан FollowLerp).
// Вызывать в игровом апдейте до рендера. Камеры без FollowID не трогаются —
// ими управляет orbit-камера редактора.
func UpdateFollowCameras(w *ecs.World, dt float32) {
	if w == nil {
		return
	}
	for _, id := range w.Entities() {
		cam, ok := w.GetCamera(id)
		if !ok || cam.FollowID == 0 {
			continue
		}
		target, hasTarget := w.GetTransform(cam.FollowID)
		if !hasTarget {
			continue
		}
		tr, hasTr := w.GetTransform(id)
		if !hasTr {
			continue
		}

		desired := target.Position.Add(cam.FollowOffset)
		if cam.FollowLerp > 0 && dt > 0 {
			// Экспоненциальное догоняние: не зависит от FPS
			t := dt * cam.FollowLerp
			if t > 1 {
				t = 1
			}
			tr.Position = emath.Lerp(tr.Position, desired, t)
		} else {
			tr.Position = desired
		}
		w.SetTransform(id, tr)
	}
}
