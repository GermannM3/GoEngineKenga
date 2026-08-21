package gameplay

import (
	"path/filepath"
	"strconv"
	"testing"

	"goenginekenga/engine/ecs"
	"goenginekenga/engine/input"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/physics"
	"goenginekenga/engine/scene"
)

// testWorld загружает сцену сэмпла CyberNinja в мир (как делает kenga run).
func testWorld(t *testing.T, sceneFile string) *ecs.World {
	t.Helper()
	path := filepath.Join("..", "..", "samples", "cyber_ninja", "scenes", sceneFile)
	s, err := scene.Load(path)
	if err != nil {
		t.Fatalf("load scene %s: %v", path, err)
	}
	return s.ToWorld()
}

// frame прогоняет один кадр: игровая логика + физика (интеграция скоростей),
// как это делает игровой цикл kenga run: systemsUpdate → rt.Step().
func frame(w *ecs.World, gls *GameLogicSystem, is *input.State) {
	dt := float32(1.0 / 60.0)
	gls.Update(w, is, dt)

	// Физика двигает тела по их скорости (те же правила, что в engine/physics)
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

// entityByName находит сущность по имени.
func entityByName(t *testing.T, w *ecs.World, name string) ecs.EntityID {
	t.Helper()
	for _, id := range w.Entities() {
		if w.Name(id) == name {
			return id
		}
	}
	t.Fatalf("entity %q not found", name)
	return 0
}

// teleport ставит сущность в позицию и обнуляет скорость.
func teleport(w *ecs.World, id ecs.EntityID, pos emath.Vec3) {
	tr, _ := w.GetTransform(id)
	tr.Position = pos
	w.SetTransform(id, tr)
	if rb, ok := w.GetRigidbody(id); ok {
		rb.Velocity = emath.V3(0, 0, 0)
		w.SetRigidbody(id, rb)
	}
}

// enemyPos возвращает текущую позицию врага (патруль двигает его по X).
func enemyPos(t *testing.T, w *ecs.World, id ecs.EntityID) emath.Vec3 {
	t.Helper()
	tr, ok := w.GetTransform(id)
	if !ok {
		t.Fatalf("enemy %v has no transform", id)
	}
	return tr.Position
}

func TestCyberNinjaEnemyPatrol(t *testing.T) {
	w := testWorld(t, "main.scene.json")
	gls := NewGameLogicSystem()
	is := input.NewState()

	enemy := entityByName(t, w, "Enemy")
	player := entityByName(t, w, "Player")

	// Уносим игрока за пределы радиуса агро — враг должен патрулировать
	teleport(w, player, emath.V3(0, 2, 50))

	// Движение: патруль от homeX=5 вправо со скоростью 2.5 ед/с (дефолтный мозг)
	tr, _ := w.GetTransform(enemy)
	startX := tr.Position.X

	for i := 0; i < 60; i++ { // 1 секунда
		frame(w, gls, is)
	}
	tr, _ = w.GetTransform(enemy)
	if tr.Position.X <= startX+0.1 {
		t.Fatalf("enemy did not patrol right: start=%v now=%v", startX, tr.Position.X)
	}
	if tr.Position.Y != 1.5 {
		t.Fatalf("enemy should float (useGravity=false), Y=%v", tr.Position.Y)
	}

	// За 5 секунд враг должен дойти до границы homeX+span=11 и развернуться
	for i := 0; i < 300; i++ {
		frame(w, gls, is)
	}
	tr, _ = w.GetTransform(enemy)
	if tr.Position.X > 11.1 {
		t.Fatalf("enemy passed patrol bound: X=%v (bound 11)", tr.Position.X)
	}
	rb, _ := w.GetRigidbody(enemy)
	if rb.Velocity.X >= 0 {
		t.Fatalf("enemy should have turned back: velocity.X=%v", rb.Velocity.X)
	}

	// Патруль не трогает игрока, который стоит в стороне
	if h, ok := w.GetHealth(player); ok && h.Current < h.Max {
		t.Fatalf("player took damage from distant enemy: %v/%v", h.Current, h.Max)
	}
}

func TestCyberNinjaEnemyDamage(t *testing.T) {
	w := testWorld(t, "main.scene.json")
	gls := NewGameLogicSystem()
	is := input.NewState()

	enemy := entityByName(t, w, "Enemy")
	player := entityByName(t, w, "Player")

	// Подносим игрока к дрону: FSM переходит в attack и бьёт по кулдауну.
	// Дефолтная атака врага: 15 урона, кулдаун 1с. 3 кадра хватает на
	// patrol→chase→attack и первый удар.
	teleport(w, player, enemyPos(t, w, enemy).Add(emath.V3(0.5, 0, 0)))
	for i := 0; i < 3; i++ {
		frame(w, gls, is)
	}
	h, _ := w.GetHealth(player)
	if h.Current != 85 {
		t.Fatalf("expected health 85 after first attack, got %v", h.Current)
	}

	// Во время неуязвимости (1с) и кулдауна врага урона нет
	for i := 0; i < 30; i++ { // ~0.5с — меньше кулдауна
		frame(w, gls, is)
	}
	h, _ = w.GetHealth(player)
	if h.Current != 85 {
		t.Fatalf("damage during invulnerability/cooldown: %v", h.Current)
	}

	// После секунды неуязвимость и кулдаун истекли — подносим игрока снова
	// (первый удар отбросил его из радиуса атаки) и ждём второй удар
	for i := 0; i < 60; i++ {
		frame(w, gls, is)
	}
	teleport(w, player, enemyPos(t, w, enemy).Add(emath.V3(0.5, 0, 0)))
	for i := 0; i < 5; i++ {
		frame(w, gls, is)
	}
	h, _ = w.GetHealth(player)
	if h.Current >= 85 {
		t.Fatalf("expected second hit after cooldown, health=%v", h.Current)
	}
}

func TestCyberNinjaPickupsAndVictory(t *testing.T) {
	w := testWorld(t, "main.scene.json")
	gls := NewGameLogicSystem()
	is := input.NewState()

	player := entityByName(t, w, "Player")

	// Собираем все 3 сферы
	for i := 0; i < 3; i++ {
		item := entityByName(t, w, "Item"+strconv.Itoa(i+1))
		itemTr, _ := w.GetTransform(item)
		teleport(w, player, itemTr.Position.Add(emath.V3(0.3, 0, 0)))
		frame(w, gls, is)

		// Предмет убран с карты после подбора
		itemTr, _ = w.GetTransform(item)
		if itemTr.Position.Y > -50 {
			t.Fatalf("item %v not removed after pickup, Y=%v", i+1, itemTr.Position.Y)
		}
	}

	if gls.score != 3 {
		t.Fatalf("score=%v, want 3", gls.score)
	}
	if gls.Status() != StatusVictory {
		t.Fatalf("status=%v, want Victory", gls.Status())
	}

	// Победа показывается в HUD
	hud := gls.HUD()
	if hud == "" {
		t.Fatal("HUD empty on victory")
	}

	// ENTER — переход на следующий уровень
	next := false
	gls.NextLevel = func() { next = true }
	is.SetKeyPressed(input.KeyEnter, true)
	frame(w, gls, is)
	if !next {
		t.Fatal("NextLevel callback not fired on ENTER after victory")
	}
}

func TestCyberNinjaDefeatAndRestart(t *testing.T) {
	w := testWorld(t, "main.scene.json")
	gls := NewGameLogicSystem()
	is := input.NewState()

	enemy := entityByName(t, w, "Enemy")
	player := entityByName(t, w, "Player")

	// Враг в состоянии attack бьёт по 15 с кулдауном 1с; держим игрока рядом,
	// пока тот не погибнет (100/15 ≈ 7 ударов ≈ 7+ секунд).
	teleport(w, player, enemyPos(t, w, enemy).Add(emath.V3(0.5, 0, 0)))
	for i := 0; i < 1200 && gls.Status() != StatusDefeat; i++ { // до 20с
		if i%60 == 0 {
			teleport(w, player, enemyPos(t, w, enemy).Add(emath.V3(0.5, 0, 0)))
		}
		frame(w, gls, is)
	}

	if gls.Status() != StatusDefeat {
		t.Fatalf("status=%v, want Defeat", gls.Status())
	}

	// R — перезапуск уровня
	restarted := false
	gls.Restart = func() { restarted = true }
	is.SetKeyPressed(input.KeyR, true)
	frame(w, gls, is)
	if !restarted {
		t.Fatal("Restart callback not fired on R")
	}
	if gls.Status() != StatusPlaying {
		t.Fatalf("status after restart=%v, want Playing", gls.Status())
	}
}
