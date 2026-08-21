package gameplay

import (
	"math"

	"goenginekenga/engine/ai"
	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
)

// FSM врага Patrol→Chase→Attack поверх engine/ai.StateMachine.
// Параметры — из компонента EnemyBrain (сцена), цель — сущность игрока.

type enemyFSM struct {
	sm    *ai.StateMachine
	agent *ai.Agent

	homeX float32 // точка спавна (центр патруля)
	homeZ float32
	dirX  float32 // направление патруля: -1/+1
}

// newEnemyFSM строит машину состояний для врага.
// update — колбэк движения: применяет velocity к rigidbody сущности.
func newEnemyFSM(w *ecs.World, id ecs.EntityID, playerID ecs.EntityID, brain ecs.EnemyBrain) *enemyFSM {
	tr, _ := w.GetTransform(id)
	f := &enemyFSM{
		agent: ai.NewAgent(tr.Position, brain.Speed),
		homeX: tr.Position.X,
		homeZ: tr.Position.Z,
		dirX:  1,
	}

	playerPos := func() emath.Vec3 { return posOf(w, playerID) }
	distToPlayer := func() float32 {
		p := playerPos()
		dx := p.X - f.agent.Position.X
		dz := p.Z - f.agent.Position.Z
		return emath.Vec3{X: dx, Y: 0, Z: dz}.Len()
	}
	move := func(vx, vz float32) {
		if rb, ok := w.GetRigidbody(id); ok {
			rb.Velocity.X = vx
			rb.Velocity.Z = vz
			w.SetRigidbody(id, rb)
		}
	}

	patrol := &ai.State{
		Name: "patrol",
		OnEnter: func(a *ai.Agent) { a.State = "patrol" },
		OnUpdate: func(a *ai.Agent, dt float32) {
			tr, ok := w.GetTransform(id)
			if !ok {
				return
			}
			// Разворот на границе зоны патруля
			if tr.Position.X > f.homeX+brain.PatrolSpan {
				f.dirX = -1
			} else if tr.Position.X < f.homeX-brain.PatrolSpan {
				f.dirX = 1
			}
			move(f.dirX*brain.Speed, 0)
			// Повернуть меш по ходу движения (forward = (sin y, -cos y))
			tr.Rotation.Y = angleToY(f.dirX, 0)
			w.SetTransform(id, tr)
			a.Position = tr.Position
		},
		Transitions: []ai.Transition{
			{Condition: func(*ai.Agent) bool { return distToPlayer() < brain.AggroRadius }, NextState: "chase"},
		},
	}

	chase := &ai.State{
		Name: "chase",
		OnEnter: func(a *ai.Agent) { a.State = "chase" },
		OnUpdate: func(a *ai.Agent, dt float32) {
			p := playerPos()
			dx := p.X - a.Position.X
			dz := p.Z - a.Position.Z
			l := emath.Vec3{X: dx, Y: 0, Z: dz}.Len()
			if l > 0.001 {
				move(dx/l*brain.ChaseSpeed, dz/l*brain.ChaseSpeed)
				// Повернуть меш в сторону игрока
				if tr, ok := w.GetTransform(id); ok {
					tr.Rotation.Y = angleToY(dx, dz)
					w.SetTransform(id, tr)
				}
			}
		},
		Transitions: []ai.Transition{
			{Condition: func(*ai.Agent) bool { return distToPlayer() <= brain.AttackRadius }, NextState: "attack"},
			{Condition: func(*ai.Agent) bool { return distToPlayer() > brain.AggroRadius*1.6 }, NextState: "patrol"},
		},
	}

	attack := &ai.State{
		Name: "attack",
		OnEnter: func(a *ai.Agent) { a.State = "attack" },
		OnUpdate: func(a *ai.Agent, dt float32) {
			move(0, 0) // стоит и бьёт
			// Урон наносит GameLogicSystem.updateEnemies по кулдауну Attacker.
		},
		Transitions: []ai.Transition{
			{Condition: func(*ai.Agent) bool { return distToPlayer() > brain.AttackRadius*1.4 }, NextState: "chase"},
		},
	}

	f.sm = ai.NewStateMachine(f.agent)
	f.sm.AddState(patrol)
	f.sm.AddState(chase)
	f.sm.AddState(attack)
	f.sm.SetState("patrol")
	return f
}

// angleToY — угол поворота вокруг Y (градусы), чтобы «вперёд» (sin, -cos) смотрел в (dx, dz).
func angleToY(dx, dz float32) float32 {
	// forward = (sin y, -cos y) = normalize(dx,dz) → y = atan2(dx, -dz)
	return float32(math.Atan2(float64(dx), float64(-dz)) * 180 / math.Pi)
}
