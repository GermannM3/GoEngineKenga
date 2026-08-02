package gameplay

import (
	"path/filepath"
	"testing"

	"goenginekenga/engine/ecs"
	"goenginekenga/engine/input"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/physics"
	"goenginekenga/engine/scene"
)

// kartFrame прогоняет один кадр гонки: KartSystem + физика (как kenga run).
func kartFrame(w *ecs.World, is *input.State) {
	dt := float32(1.0 / 60.0)
	KartSystem(w, is, dt)

	ids := w.Entities()
	bodies := make([]*physics.Rigidbody, 0, len(ids))
	transforms := make([]emath.Vec3, 0, len(ids))
	alive := make([]ecs.EntityID, 0, len(ids))
	for _, id := range ids {
		rb, ok1 := w.GetRigidbody(id)
		tr, ok2 := w.GetTransform(id)
		if !ok1 || !ok2 {
			continue
		}
		bodies = append(bodies, &rb)
		transforms = append(transforms, tr.Position)
		alive = append(alive, id)
	}
	physics.DefaultPhysicsWorld().Update(dt, bodies, nil, transforms)
	for i, id := range alive {
		tr, _ := w.GetTransform(id)
		tr.Position = transforms[i]
		w.SetTransform(id, tr)
		w.SetRigidbody(id, *bodies[i])
	}
}

func kartWorld(t *testing.T) *ecs.World {
	t.Helper()
	path := filepath.Join("..", "..", "samples", "kart_racing", "scenes", "race.scene.json")
	s, err := scene.Load(path)
	if err != nil {
		t.Fatalf("load scene %s: %v", path, err)
	}
	return s.ToWorld()
}

// waitRacing прогоняет кадры, пока гонка не перейдёт в фазу Racing.
func waitRacing(t *testing.T, w *ecs.World, is *input.State) {
	t.Helper()
	for i := 0; i < 300 && GetRaceState().Phase != PhaseRacing; i++ {
		kartFrame(w, is)
	}
	if GetRaceState().Phase != PhaseRacing {
		t.Fatalf("race did not start, phase=%v", GetRaceState().Phase)
	}
}

func TestKartRaceFlow(t *testing.T) {
	w := kartWorld(t)
	is := input.NewState()
	rs := GetRaceState()

	// Меню: выбор машины клавишей 2
	ResetRace(w) // сразу к countdown — тестируем саму гонку
	if rs.Phase != PhaseCountdown {
		t.Fatalf("phase=%v, want Countdown", rs.Phase)
	}

	waitRacing(t, w, is)
	kartFrame(w, is)
	kartFrame(w, is)
	if rs.Elapsed() <= 0 {
		t.Fatal("RaceTime not accumulating")
	}

	// Игрок проезжает 3 круга по чекпоинтам (16 вейпоинтов на круг)
	player := entityByName(t, w, "Player")
	wps := WaypointsForDraw()
	if len(wps) != 16 {
		t.Fatalf("waypoints=%d, want 16", len(wps))
	}
	lapsSeen := 0
	guard := 0
	for lapsSeen < LapsToWin && guard < 400 {
		guard++
		for _, wp := range wps {
			teleport(w, player, wp)
			kartFrame(w, is)
		}
		kart, _ := w.GetKart(player)
		lapsSeen = kart.CurrentLap
	}

	kart, _ := w.GetKart(player)
	if kart.CurrentLap < LapsToWin {
		t.Fatalf("player laps=%d, want %d", kart.CurrentLap, LapsToWin)
	}
	if kart.LastLapTime <= 0 {
		t.Fatalf("LastLapTime=%v, want >0", kart.LastLapTime)
	}
	if kart.BestLapTime <= 0 || kart.BestLapTime > kart.LastLapTime+0.001 {
		t.Fatalf("BestLapTime=%v, LastLapTime=%v", kart.BestLapTime, kart.LastLapTime)
	}

	// Финиш: игрок завершил гонку
	if rs.Phase != PhaseFinished {
		t.Fatalf("phase=%v, want Finished after %d laps", rs.Phase, lapsSeen)
	}
	_, msg, _, winner := rs.Snapshot()
	if winner != "Player" {
		t.Fatalf("winner=%q, want Player", winner)
	}
	if msg == "" {
		t.Fatal("finish message empty")
	}
}

func TestKartBotRaces(t *testing.T) {
	w := kartWorld(t)
	is := input.NewState()
	rs := GetRaceState()

	ResetRace(w)
	waitRacing(t, w, is)

	// Боты едут по вейпоинтам: за 10 секунд симуляции они должны проехать
	// хотя бы несколько чекпоинтов и набрать прогресс.
	bot := entityByName(t, w, "Bot1")
	start := progressOf(t, w, bot)
	for i := 0; i < 600; i++ {
		kartFrame(w, is)
	}
	after := progressOf(t, w, bot)
	if after <= start {
		t.Fatalf("bot did not progress: %v -> %v", start, after)
	}
	if rs.Phase != PhaseRacing {
		t.Fatalf("phase=%v, bots should still race (10s < lap time)", rs.Phase)
	}
}

// progressOf — прогресс карта: круг*16 + чекпоинт.
func progressOf(t *testing.T, w *ecs.World, id ecs.EntityID) float32 {
	t.Helper()
	kart, ok := w.GetKart(id)
	if !ok {
		t.Fatalf("no kart on entity %v", id)
	}
	return float32(kart.CurrentLap)*16 + float32(kart.LastCheckpoint)
}
