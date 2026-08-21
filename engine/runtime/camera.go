package runtime

import (
	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
)

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
