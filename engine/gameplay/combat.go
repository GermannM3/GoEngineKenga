package gameplay

import (
	"math"

	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/input"
)

// Ближний бой: Attacker (кулдаун, сектор удара) + Health (урон, i-frames)
// + knockback через Rigidbody. Игрок атакует по вводу, враги — из своего FSM.

const (
	playerInvulnAfterHit = 1.0 // сек неуязвимости игрока после урона
	enemyInvulnAfterHit  = 0.3 // сек неуязвимости врага между ударами
)

// ForwardOf возвращает горизонтальное направление «вперёд» по rotation.Y.
// Конвенция движка (см. orbit_camera.go): rotY=0 → смотрит в -Z.
func ForwardOf(tr ecs.Transform) emath.Vec3 {
	rad := float64(tr.Rotation.Y) * math.Pi / 180
	return emath.Vec3{X: float32(math.Sin(rad)), Y: 0, Z: float32(-math.Cos(rad))}
}

// tickAttackers продвигает кулдауны и таймеры неуязвимости всех бойцов.
func tickAttackers(w *ecs.World, dt float32) {
	for _, id := range w.Entities() {
		if a, ok := w.GetAttacker(id); ok && a.Timer > 0 {
			a.Timer -= dt
			w.SetAttacker(id, a)
		}
		if h, ok := w.GetHealth(id); ok && h.InvulnTimer > 0 {
			h.InvulnTimer -= dt
			w.SetHealth(id, h)
		}
	}
}

// attackTargetsOf — цели удара attacker'а: сущности со здоровьем в радиусе
// и внутри сектора атаки (ArcDeg), исключая самого атакующего.
func attackTargetsOf(w *ecs.World, attackerID ecs.EntityID, tr ecs.Transform, a ecs.Attacker) []ecs.EntityID {
	fwd := ForwardOf(tr)
	halfArc := a.ArcDeg / 2 * math.Pi / 180
	if halfArc <= 0 { // не настроен — круговой удар
		halfArc = math.Pi
	}

	var targets []ecs.EntityID
	for _, id := range w.Entities() {
		if id == attackerID {
			continue
		}
		if _, hasHP := w.GetHealth(id); !hasHP {
			continue // бьём только то, что может умереть
		}
		tTr, ok := w.GetTransform(id)
		if !ok {
			continue
		}
		to := tTr.Position.Sub(tr.Position)
		distXZ := emath.Vec3{X: to.X, Y: 0, Z: to.Z}.Len()
		if distXZ > a.Radius || absF(to.Y) > a.Radius {
			continue
		}
		if distXZ > 0.001 { // вплотную — считаем целью всегда
			dot := (fwd.X*to.X + fwd.Z*to.Z) / distXZ
			if dot < float32(math.Cos(float64(halfArc))) {
				continue // вне сектора
			}
		}
		targets = append(targets, id)
	}
	return targets
}

// applyHit наносит урон цели: здоровье, i-frames, отброс. Возвращает true, если цель умерла.
func applyHit(w *ecs.World, targetID ecs.EntityID, dmg, knockback float32, from emath.Vec3) bool {
	h, ok := w.GetHealth(targetID)
	if !ok || h.InvulnTimer > 0 || h.Current <= 0 {
		return false
	}
	h.Current -= dmg
	if h.Current < 0 {
		h.Current = 0
	}
	h.InvulnTimer = enemyInvulnAfterHit
	w.SetHealth(targetID, h)

	// Отброс от источника удара
	if rb, hasRb := w.GetRigidbody(targetID); hasRb {
		tpos := posOf(w, targetID)
		dir := emath.Vec3{X: tpos.X - from.X, Y: 0, Z: tpos.Z - from.Z}
		if l := dir.Len(); l > 0.001 {
			dir = dir.Mul(1 / l)
		} else {
			dir = emath.Vec3{X: 0, Y: 0, Z: 1}
		}
		rb.Velocity.X = dir.X * knockback
		rb.Velocity.Z = dir.Z * knockback
		if rb.Velocity.Y < knockback*0.5 {
			rb.Velocity.Y = knockback * 0.5
		}
		w.SetRigidbody(targetID, rb)
	}
	return h.Current <= 0
}

// playerWantsAttack — нажата ли клавиша/кнопка атаки (F или ЛКМ).
func playerWantsAttack(is *input.State) bool {
	if is == nil {
		return false
	}
	return is.IsKeyJustPressed(input.KeyF) || is.IsMouseButtonJustPressed(input.MouseButtonLeft)
}

// posOf — позиция сущности (нулевая, если transform нет).
func posOf(w *ecs.World, id ecs.EntityID) emath.Vec3 {
	if tr, ok := w.GetTransform(id); ok {
		return tr.Position
	}
	return emath.Vec3{}
}

func absF(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
