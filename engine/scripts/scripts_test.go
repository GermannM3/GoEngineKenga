package scripts

import (
	"testing"

	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/scene"
)

// counterScript считает вызовы жизненного цикла.
type counterScript struct {
	starts, updates int
	speed           float32
	lastX           float32
}

func (c *counterScript) OnStart(ctx *Context) {
	c.starts++
	c.speed = ctx.Float("speed", 1)
}

func (c *counterScript) OnUpdate(ctx *Context, dt float32) {
	c.updates++
	c.lastX = posOfX(ctx)
}

func posOfX(ctx *Context) float32 {
	if tr, ok := ctx.World.GetTransform(ctx.Entity); ok {
		return tr.Position.X
	}
	return 0
}

// mover двигает объект по X со скоростью из параметров (для проверки эффектов).
type mover struct{}

func (m *mover) OnStart(ctx *Context) {}

func (m *mover) OnUpdate(ctx *Context, dt float32) {
	tr, ok := ctx.World.GetTransform(ctx.Entity)
	if !ok {
		return
	}
	tr.Position.X += dt * ctx.Float("speed", 10)
	ctx.World.SetTransform(ctx.Entity, tr)
}

func testWorld(t *testing.T) (*ecs.World, ecs.EntityID) {
	t.Helper()
	s := &scene.Scene{Name: "t", Entities: []scene.SceneEntity{
		{
			Name:      "Box",
			Transform: &ecs.Transform{Position: emath.Vec3{X: 0, Y: 1, Z: 0}, Scale: emath.Vec3{X: 1, Y: 1, Z: 1}},
			Script:    &ecs.Script{Name: "counter", Params: map[string]string{"speed": "5"}},
		},
	}}
	w := s.ToWorld()
	for _, id := range w.Entities() {
		if w.Name(id) == "Box" {
			return w, id
		}
	}
	t.Fatal("entity not found")
	return nil, 0
}

func resetRegistry() {
	mu.Lock()
	defer mu.Unlock()
	registry = map[string]func() Script{}
}

func TestLifecycleStartOnceThenUpdates(t *testing.T) {
	resetRegistry()
	c := &counterScript{}
	Register("counter", func() Script { return c })

	w, _ := testWorld(t)
	sys := NewSystem(w)
	sys.Update(0.1)
	sys.Update(0.1)
	sys.Update(0.1)

	if c.starts != 1 {
		t.Fatalf("OnStart вызван %d раз, ожидался 1", c.starts)
	}
	if c.updates != 3 {
		t.Fatalf("OnUpdate вызван %d раз, ожидалось 3", c.updates)
	}
	if c.speed != 5 {
		t.Fatalf("параметр speed не дошёл: %f, ожидалось 5", c.speed)
	}
}

func TestMoverMovesEntity(t *testing.T) {
	resetRegistry()
	Register("mover", func() Script { return &mover{} })

	s := &scene.Scene{Name: "t", Entities: []scene.SceneEntity{
		{
			Name:      "M",
			Transform: &ecs.Transform{Position: emath.Vec3{X: 0, Y: 0, Z: 0}, Scale: emath.Vec3{X: 1, Y: 1, Z: 1}},
			Script:    &ecs.Script{Name: "mover", Params: map[string]string{"speed": "1"}},
		},
	}}
	w := s.ToWorld()
	var id ecs.EntityID
	for _, e := range w.Entities() {
		id = e
	}

	sys := NewSystem(w)
	for i := 0; i < 30; i++ {
		sys.Update(1.0 / 30)
	}
	tr, _ := w.GetTransform(id)
	if tr.Position.X < 0.9 || tr.Position.X > 1.1 {
		t.Fatalf("скрипт не двигал объект: X=%f, ожидалось ~1.0", tr.Position.X)
	}
}

func TestAttachWorldResetsState(t *testing.T) {
	resetRegistry()
	c := &counterScript{}
	Register("counter", func() Script { return c })

	w, _ := testWorld(t)
	sys := NewSystem(w)
	sys.Update(0.1)
	sys.Update(0.1)

	// Новый мир (новый уровень/hot-reload): инстансы сбрасываются
	w2 := w.Clone()
	sys.AttachWorld(w2)
	sys.Update(0.1)

	if c.starts != 2 {
		t.Fatalf("после смены мира OnStart должен вызваться заново: %d", c.starts)
	}
}

func TestUnknownScriptIsSkipped(t *testing.T) {
	resetRegistry()
	s := &scene.Scene{Name: "t", Entities: []scene.SceneEntity{
		{Name: "Ghost", Script: &ecs.Script{Name: "no_such_script"}},
	}}
	w := s.ToWorld()
	sys := NewSystem(w)
	sys.Update(0.1) // не должно паниковать
}

