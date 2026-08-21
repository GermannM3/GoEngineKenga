package gameplay

import (
	"math/rand"

	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
)

// Визуальные эффекты боя: осколки-частицы как сущности. Используют обычный
// меш-pipeline (кристалл), поэтому работают в обоих бэкендах без нового GPU-кода.
// ponytail: настоящие GPU-частицы (инстансированные билборды) — апгрейд,
// когда осколков станет > сотни на экране.

const shardLifetime = 0.7 // сек жизни осколка

// SpawnBurst спавнит n осколков, разлетающихся из точки pos.
// shardMeshID — asset ID меша осколка ("" — осколки не спавнятся).
func SpawnBurst(w *ecs.World, shardMeshID string, pos emath.Vec3, n int) {
	if w == nil || shardMeshID == "" {
		return
	}
	for i := 0; i < n; i++ {
		id := w.CreateEntity("shard")
		dir := emath.Vec3{
			X: rand.Float32()*2 - 1,
			Y: rand.Float32()*0.9 + 0.3, // вверх и в стороны
			Z: rand.Float32()*2 - 1,
		}
		if l := dir.Len(); l > 0.001 {
			dir = dir.Mul(1 / l)
		}
		w.SetTransform(id, ecs.Transform{
			Position: pos.Add(emath.Vec3{X: rand.Float32()*0.3 - 0.15, Y: rand.Float32()*0.3 - 0.15, Z: rand.Float32()*0.3 - 0.15}),
			Rotation: emath.Vec3{X: rand.Float32() * 360, Y: rand.Float32() * 360, Z: rand.Float32() * 360},
			Scale:    emath.Vec3{X: 0.18, Y: 0.28, Z: 0.18},
		})
		w.SetMeshRenderer(id, ecs.MeshRenderer{
			MeshAssetID: shardMeshID,
			ColorR:      255, ColorG: 225, ColorB: 130, ColorA: 255,
		})
		speed := float32(2.5) + rand.Float32()*3
		w.SetRigidbody(id, ecs.Rigidbody{
			Mass: 0.2, UseGravity: true, Drag: 0.5,
			Velocity: dir.Mul(speed),
		})
	}
}

// updateShards затухает и удаляет осколки; вызывается из Update.
func (gls *GameLogicSystem) updateShards(w *ecs.World, dt float32) {
	for id, life := range gls.shards {
		life -= dt
		if life <= 0 {
			w.RemoveEntity(id)
			delete(gls.shards, id)
			continue
		}
		// Сжимаем осколок по мере затухания
		if tr, ok := w.GetTransform(id); ok {
			k := life / shardLifetime
			tr.Scale = emath.Vec3{X: 0.18 * k, Y: 0.28 * k, Z: 0.18 * k}
			w.SetTransform(id, tr)
		}
		gls.shards[id] = life
	}
}

// trackShards регистрирует новые осколки (сущности "shard" без учёта).
func (gls *GameLogicSystem) trackShards(w *ecs.World) {
	for _, id := range w.Entities() {
		if w.Name(id) == "shard" {
			if _, ok := gls.shards[id]; !ok {
				gls.shards[id] = shardLifetime
			}
		}
	}
}

// burst спавнит осколки и берёт их на учёт (хелпер для событий боя).
func (gls *GameLogicSystem) burst(w *ecs.World, pos emath.Vec3, n int) {
	SpawnBurst(w, gls.ShardMeshID, pos, n)
	gls.trackShards(w)
}
