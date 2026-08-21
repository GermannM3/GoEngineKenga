package gameplay

import (
	"testing"

	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
)

func TestShardBurstLifecycle(t *testing.T) {
	w := combatScene().ToWorld()
	gls := newTestGLS()
	gls.ShardMeshID = "test-mesh-id"
	before := len(w.Entities())

	SpawnBurst(w, "test-mesh-id", emath.V3(0, 1, 0), 6)
	gls.trackShards(w)
	if got := len(w.Entities()) - before; got != 6 {
		t.Fatalf("осколков заспавнено %d, ожидалось 6", got)
	}
	if len(gls.shards) != 6 {
		t.Fatalf("на учёте %d осколков, ожидалось 6", len(gls.shards))
	}

	// Осколок имеет меш и гравитацию (падает вниз)
	var shard ecs.EntityID
	for _, id := range w.Entities() {
		if w.Name(id) == "shard" {
			shard = id
			break
		}
	}
	if mr, ok := w.GetMeshRenderer(shard); !ok || mr.MeshAssetID != "test-mesh-id" {
		t.Fatal("у осколка нет меша")
	}
	if rb, ok := w.GetRigidbody(shard); !ok || !rb.UseGravity {
		t.Fatal("у осколка нет гравитации")
	}

	// За секунду жизни осколки исчезают
	for i := 0; i < 60; i++ {
		gls.Update(w, nil, 1.0/60)
	}
	if len(gls.shards) != 0 {
		t.Fatalf("осколки не удалены за 1с: осталось %d", len(gls.shards))
	}
	if after := len(w.Entities()); after > before {
		t.Fatalf("сущности-осколки остались в мире: было %d, стало %d", before, after)
	}
}

func TestShardBurstWithoutMesh(t *testing.T) {
	w := combatScene().ToWorld()
	before := len(w.Entities())
	SpawnBurst(w, "", emath.V3(0, 1, 0), 5)
	if len(w.Entities()) != before {
		t.Fatal("без меша осколки спавниться не должны")
	}
}
