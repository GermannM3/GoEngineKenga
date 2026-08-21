package gameplay

import (
	"testing"

	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/input"
	"goenginekenga/engine/scene"
)

// combatScene: игрок с атакой и враг с здоровьем на расстоянии 1.5 по -Z
// (игрок с rotY=0 смотрит в -Z — враг прямо перед ним).
func combatScene() *scene.Scene {
	return &scene.Scene{
		Name: "combat",
		Entities: []scene.SceneEntity{
			{
				Name:      "Player",
				Transform: &ecs.Transform{Position: emath.V3(0, 1, 0), Scale: emath.V3(1, 1, 1)},
				Rigidbody: &ecs.Rigidbody{Mass: 1},
				Health:    &ecs.Health{Current: 100, Max: 100},
				Attacker:  &ecs.Attacker{Radius: 2.2, ArcDeg: 120, Damage: 34, Cooldown: 0.45, Knockback: 7},
			},
			{
				Name:      "Enemy1",
				Transform: &ecs.Transform{Position: emath.V3(0, 1, -1.5), Scale: emath.V3(1, 1, 1)},
				Rigidbody: &ecs.Rigidbody{Mass: 1},
				Health:    &ecs.Health{Current: 50, Max: 50},
				Attacker:  &ecs.Attacker{Radius: 1.8, ArcDeg: 360, Damage: 15, Cooldown: 1.0, Knockback: 5},
				EnemyBrain: &ecs.EnemyBrain{
					PatrolSpan: 3, Speed: 1, ChaseSpeed: 2, AggroRadius: 8, AttackRadius: 1.6,
				},
			},
		},
	}
}

func newTestGLS() *GameLogicSystem {
	gls := NewGameLogicSystem()
	gls.SetLevel(1)
	return gls
}

func pressKeyState(k input.Key) *input.State {
	st := input.NewState()
	st.SetKeyPressed(k, true)
	return st
}

func TestPlayerAttackKillsEnemy(t *testing.T) {
	w := combatScene().ToWorld()
	gls := newTestGLS()

	// Первый Update инициализирует состояние (readHealth), затем удар
	gls.Update(w, nil, 1.0/60)
	if gls.health != 100 {
		t.Fatalf("health не прочитан из компонента: %f", gls.health)
	}

	// Первый удар: 50 - 34 = 16 HP, враг жив
	gls.Update(w, pressKeyState(input.KeyF), 1.0/60)
	if h, ok := w.GetHealth(gls.enemyIDs[0]); !ok {
		t.Fatal("враг удалён раньше времени")
	} else if h.Current != 16 {
		t.Fatalf("урон не применился: HP=%f, ожидалось 16", h.Current)
	}
	if gls.kills != 0 {
		t.Fatalf("убийство засчитано раньше времени: %d", gls.kills)
	}

	// Кулдаун не даёт бить сразу
	gls.Update(w, nil, 0.1)
	gls.Update(w, pressKeyState(input.KeyF), 1.0/60)
	if h, ok := w.GetHealth(gls.enemyIDs[0]); ok && h.Current != 16 {
		t.Fatalf("удар прошёл сквозь кулдаун: HP=%f", h.Current)
	}

	// Дождались кулдауна — добили
	gls.Update(w, nil, 0.5)
	gls.Update(w, pressKeyState(input.KeyF), 1.0/60)
	if _, still := w.GetHealth(gls.enemyIDs[0]); still {
		t.Fatal("враг должен быть удалён после смертельного удара")
	}
	if gls.kills != 1 {
		t.Fatalf("убийство не засчитано: kills=%d", gls.kills)
	}
}

func TestAttackArcBlocksBackstab(t *testing.T) {
	w := combatScene().ToWorld()
	// Врага за спиной игрока (игрок смотрит в -Z, враг в +Z)
	tr, _ := w.GetTransform(w.Entities()[1]) // Enemy1
	tr.Position = emath.V3(0, 1, 1.5)
	w.SetTransform(w.Entities()[1], tr)

	gls := newTestGLS()
	gls.Update(w, nil, 1.0/60) // init
	gls.Update(w, pressKeyState(input.KeyF), 1.0/60)

	if h, ok := w.GetHealth(gls.enemyIDs[0]); !ok || h.Current != 50 {
		t.Fatalf("удар в спину не должен проходить: HP=%v", h)
	}
}

func TestEnemyFSMChasesAndAttacks(t *testing.T) {
	w := combatScene().ToWorld()
	gls := newTestGLS()
	gls.Update(w, nil, 1.0/60) // init

	// Игрок стоит рядом (в радиусе атаки 1.6): враг переходит в attack
	// и бьёт по кулдауну 1с. За 3 секунды должно быть >= 2 ударов по 15.
	for i := 0; i < 180; i++ {
		gls.Update(w, nil, 1.0/60)
	}
	if got := 100 - gls.health; got < 25 {
		t.Fatalf("враг в состоянии attack почти не бьёт: урон=%f за 3с", got)
	}
}

func TestEnemyPatrolsWhenPlayerFar(t *testing.T) {
	w := combatScene().ToWorld()
	// Уносим игрока за пределы агро
	tr, _ := w.GetTransform(w.Entities()[0])
	tr.Position = emath.V3(0, 1, 50)
	w.SetTransform(w.Entities()[0], tr)

	gls := newTestGLS()
	gls.Update(w, nil, 1.0/60) // init

	fsm := gls.enemyFSMs[gls.enemyIDs[0]]
	if fsm == nil || fsm.agent.State != "patrol" {
		t.Fatalf("враг должен патрулировать, состояние: %+v", fsm)
	}
	// Патруль задаёт скорость движения по X
	rb, _ := w.GetRigidbody(gls.enemyIDs[0])
	if rb.Velocity.X != 1 { // Speed из EnemyBrain
		t.Fatalf("патруль не двигается: velocity.X=%f, ожидалось 1", rb.Velocity.X)
	}
}

func TestIFramesBlockDoubleHit(t *testing.T) {
	w := combatScene().ToWorld()
	gls := newTestGLS()
	gls.Update(w, nil, 1.0/60) // init

	// Даём врагу побольше HP, чтобы пережить два удара
	h, _ := w.GetHealth(gls.enemyIDs[0])
	h.Current = 100
	w.SetHealth(gls.enemyIDs[0], h)

	gls.Update(w, pressKeyState(input.KeyF), 1.0/60) // 100→66, i-frames 0.3с
	// Сбрасываем кулдаун атакующего, но i-frames ещё держатся — удар блокируется
	a, _ := w.GetAttacker(gls.playerID)
	a.Timer = 0
	w.SetAttacker(gls.playerID, a)
	gls.Update(w, pressKeyState(input.KeyF), 1.0/60)

	if h, _ := w.GetHealth(gls.enemyIDs[0]); h.Current != 66 {
		t.Fatalf("i-frames не сработали: HP=%f, ожидалось 66", h.Current)
	}
}
