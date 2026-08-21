package runtime

import (
	"testing"

	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/input"
	"goenginekenga/engine/scene"
)

// testControllerScene: Player (капсула) над статичным полом-боксом (верх на y=0).
func testControllerScene() *scene.Scene {
	return &scene.Scene{
		Name: "test",
		Entities: []scene.SceneEntity{
			{
				Name:      "Player",
				Transform: &ecs.Transform{Position: emath.V3(0, 3, 0), Scale: emath.V3(1, 1, 1)},
				Rigidbody: &ecs.Rigidbody{Mass: 1, UseGravity: true, Drag: 2},
				Collider:  &ecs.Collider{Type: "capsule", Radius: 0.5, Height: 1.8},
				CharacterController: &ecs.CharacterController{
					MoveSpeed: 6, JumpForce: 9,
				},
			},
			{
				Name:      "Ground",
				Transform: &ecs.Transform{Position: emath.V3(0, -0.5, 0), Scale: emath.V3(1, 1, 1)},
				Collider:  &ecs.Collider{Type: "box", Size: emath.V3(20, 1, 20)},
			},
		},
	}
}

// pressKey готовит состояние ввода с «только что нажатой» клавишей.
func pressKey(k input.Key) *input.State {
	st := input.NewState()
	st.SetKeyPressed(k, true)
	return st
}

func TestCharacterControllerLands(t *testing.T) {
	rt := NewFromScene(testControllerScene())
	rt.StartPlay()
	w, _ := rt.ActiveWorld()
	playerID := findEntity(w, "Player")

	grounded := false
	for i := 0; i < 240; i++ { // до 4 секунд симуляции
		rt.Step()
		cc, _ := w.GetCharacterController(playerID)
		if cc.Grounded {
			grounded = true
			break
		}
	}
	if !grounded {
		t.Fatal("игрок не приземлился за 4 секунды: Grounded так и не стал true")
	}

	tr, _ := w.GetTransform(playerID)
	// Низ капсулы (r=0.5, h=1.8 → низ = центр - 1.4) должен быть у пола y=0
	want := float32(1.4)
	if tr.Position.Y < want-0.35 || tr.Position.Y > want+0.35 {
		t.Fatalf("игрок завис не на полу: Y=%f, ожидалось ~%f", tr.Position.Y, want)
	}
}

func TestJumpOnlyWhenGrounded(t *testing.T) {
	rt := NewFromScene(testControllerScene())
	rt.StartPlay()
	w, _ := rt.ActiveWorld()
	playerID := findEntity(w, "Player")

	// Прыжок в воздухе (сразу, игрок ещё падает) не должен подбросить
	ApplyPlayerInput(w, pressKey(input.KeySpace), 1.0/60)
	rb, _ := w.GetRigidbody(playerID)
	if rb.Velocity.Y > 0.001 && rb.Velocity.Y >= 9-0.001 {
		t.Fatalf("прыжок сработал в воздухе: velocity.Y=%f", rb.Velocity.Y)
	}

	// Приземляемся
	for i := 0; i < 240; i++ {
		rt.Step()
		cc, _ := w.GetCharacterController(playerID)
		if cc.Grounded {
			break
		}
	}

	// Теперь прыжок разрешён
	ApplyPlayerInput(w, pressKey(input.KeySpace), 1.0/60)
	rb, _ = w.GetRigidbody(playerID)
	if rb.Velocity.Y < 9 {
		t.Fatalf("прыжок с земли не сработал: velocity.Y=%f, ожидалось >=9", rb.Velocity.Y)
	}
}

func TestMoveSpeedFromController(t *testing.T) {
	rt := NewFromScene(testControllerScene())
	rt.StartPlay()
	w, _ := rt.ActiveWorld()

	ApplyPlayerInput(w, func() *input.State {
		st := input.NewState()
		st.SetKeyPressed(input.KeyD, true)
		return st
	}(), 1.0/60)

	rb, _ := w.GetRigidbody(findEntity(w, "Player"))
	if rb.Velocity.X != 6 {
		t.Fatalf("скорость движения из CharacterController не применилась: X=%f, ожидалось 6", rb.Velocity.X)
	}
}

func findEntity(w *ecs.World, name string) ecs.EntityID {
	for _, id := range w.Entities() {
		if w.Name(id) == name {
			return id
		}
	}
	return 0
}
