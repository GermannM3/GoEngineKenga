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
// Параметры движения берутся из CharacterController (если есть), прыжок разрешён
// только когда контроллер стоит на земле.
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

	moveSpeed := float32(playerMoveSpeed)
	jumpForce := float32(playerJumpForce)
	canJump := rb.Velocity.Y <= 0 // без контроллера — старая эвристика
	cc, hasCC := w.GetCharacterController(playerID)
	if hasCC {
		if cc.MoveSpeed > 0 {
			moveSpeed = cc.MoveSpeed
		}
		if cc.JumpForce > 0 {
			jumpForce = cc.JumpForce
		}
		canJump = cc.Grounded
	}

	// Горизонтальное движение: клавиатура или левый стик/крестовина геймпада
	vx := rb.Velocity.X
	const padDeadzone = 0.2
	padX := input.Deadzone(st.PadAxisValue(input.PadAxisLX), padDeadzone)
	switch {
	case st.IsKeyPressed(input.KeyA) || st.IsKeyPressed(input.KeyArrowLeft) || st.IsPadButtonDown(input.PadLeft):
		vx = -moveSpeed
	case st.IsKeyPressed(input.KeyD) || st.IsKeyPressed(input.KeyArrowRight) || st.IsPadButtonDown(input.PadRight):
		vx = moveSpeed
	default:
		if padX != 0 {
			vx = moveSpeed * padX // аналоговая скорость по наклону стика
		} else {
			vx = vx * (1.0 - rb.Drag*dt) // затухание
			if vx > -0.1 && vx < 0.1 {
				vx = 0
			}
		}
	}
	rb.Velocity.X = vx

	// Прыжок: только с земли; Space/W/↑ или кнопка A геймпада
	jumpPressed := st.IsKeyJustPressed(input.KeySpace) ||
		st.IsKeyJustPressed(input.KeyW) ||
		st.IsKeyJustPressed(input.KeyArrowUp) ||
		st.IsPadButtonJustPressed(input.PadA)
	if canJump && jumpPressed {
		rb.Velocity.Y = jumpForce
		if hasCC {
			cc.Grounded = false // отрываемся до следующего физического шага
			w.SetCharacterController(playerID, cc)
		}
	}

	w.SetRigidbody(playerID, rb)
}
