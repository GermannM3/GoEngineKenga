package runtime

import (
	"goenginekenga/engine/ecs"
	"goenginekenga/engine/input"
)

const (
	playerMoveSpeed = 6.0
	playerJumpForce = 9.0
)

// ApplyPlayerInput применяет ввод к сущности с именем "Player": A/D или стрелки — движение, Space — прыжок.
// Вызывать до rt.Step(), чтобы скорость попала в физику. inputState — *input.State или nil.
func ApplyPlayerInput(w *ecs.World, inputState interface{}, dt float32) {
	if w == nil || inputState == nil {
		return
	}
	st, ok := inputState.(*input.State)
	if !ok {
		return
	}

	var playerID ecs.EntityID
	var found bool
	for _, id := range w.Entities() {
		if w.Name(id) == "Player" {
			playerID = id
			found = true
			break
		}
	}
	if !found {
		return
	}

	rb, ok := w.GetRigidbody(playerID)
	if !ok {
		return
	}

	// Горизонтальное движение (X)
	vx := rb.Velocity.X
	if st.IsKeyPressed(input.KeyA) || st.IsKeyPressed(input.KeyArrowLeft) {
		vx = -playerMoveSpeed
	} else if st.IsKeyPressed(input.KeyD) || st.IsKeyPressed(input.KeyArrowRight) {
		vx = playerMoveSpeed
	} else {
		vx = vx * (1.0 - rb.Drag*dt) // затухание
		if vx > -0.1 && vx < 0.1 {
			vx = 0
		}
	}
	rb.Velocity.X = vx

	// Прыжок: только если уже падаем или стоим (velocity.Y <= 0), и Space нажато в этом кадре
	if (st.IsKeyJustPressed(input.KeySpace) || st.IsKeyJustPressed(input.KeyW) || st.IsKeyJustPressed(input.KeyArrowUp)) && rb.Velocity.Y <= 0 {
		rb.Velocity.Y = playerJumpForce
	}

	w.SetRigidbody(playerID, rb)
}
