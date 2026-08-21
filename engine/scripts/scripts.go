// Package scripts — скриптовые компоненты сущностей (аналог MonoBehaviour).
//
// Скрипт регистрируется по имени через Register (обычно в init() пакета игры),
// а сцена ссылается на него компонентом ecs.Script{Name, Params}. Так игровая
// логика живёт в проекте игры, а не в движке: kenga.exe остаётся универсальным
// раннером, а игра со своими Go-скриптами собирается как свой exe, который
// импортирует пакеты со скриптами (см. samples/cyber_ninja/scripts).
package scripts

import (
	"sort"
	"strconv"
	"sync"

	"goenginekenga/engine/ecs"
)

// Script — поведение сущности. OnStart вызывается один раз перед первым
// апдейтом, OnUpdate — каждый кадр. Реализуется игрой.
type Script interface {
	OnStart(ctx *Context)
	OnUpdate(ctx *Context, dt float32)
}

// Context — окружение скрипта: мир, его сущность и параметры из сцены.
type Context struct {
	World  *ecs.World
	Entity ecs.EntityID
	Params map[string]string
}

// Float возвращает числовой параметр (или def, если нет/не число).
func (c *Context) Float(key string, def float32) float32 {
	if v, ok := c.Params[key]; ok {
		if f, err := strconv.ParseFloat(v, 32); err == nil {
			return float32(f)
		}
	}
	return def
}

// String возвращает строковый параметр (или def).
func (c *Context) String(key, def string) string {
	if v, ok := c.Params[key]; ok && v != "" {
		return v
	}
	return def
}

var (
	mu       sync.RWMutex
	registry = map[string]func() Script{}
)

// Register регистрирует фабрику скрипта под именем (для ecs.Script.Name).
// Вызывать из init() пакета игры.
func Register(name string, factory func() Script) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = factory
}

// Create создаёт экземпляр скрипта по имени; nil — имя не зарегистрировано.
func Create(name string) Script {
	mu.RLock()
	defer mu.RUnlock()
	if f, ok := registry[name]; ok {
		return f()
	}
	return nil
}

// Names возвращает отсортированные имена зарегистрированных скриптов.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// System ведёт жизненный цикл скриптов мира: Start один раз, Update каждый кадр.
type System struct {
	world    *ecs.World
	started  map[ecs.EntityID]bool
	insts    map[ecs.EntityID]Script
	ctx      Context
	warnings map[ecs.EntityID]string // имя незарегистрированного скрипта (лог раз)
}

// NewSystem привязывает систему к миру.
func NewSystem(w *ecs.World) *System {
	return &System{
		world:    w,
		started:  map[ecs.EntityID]bool{},
		insts:    map[ecs.EntityID]Script{},
		warnings: map[ecs.EntityID]string{},
	}
}

// AttachWorld перепривязывает систему к новому миру (hot-reload, новый уровень):
// инстансы и состояние старта сбрасываются.
func (s *System) AttachWorld(w *ecs.World) {
	if s.world == w {
		return
	}
	s.world = w
	s.started = map[ecs.EntityID]bool{}
	s.insts = map[ecs.EntityID]Script{}
	s.warnings = map[ecs.EntityID]string{}
}

// Update продвигает все скрипты мира на dt.
func (s *System) Update(dt float32) {
	if s.world == nil {
		return
	}
	for _, id := range s.world.Entities() {
		sc, ok := s.world.GetScript(id)
		if !ok || sc.Name == "" {
			continue
		}
		inst, ok := s.insts[id]
		if !ok {
			inst = Create(sc.Name)
			if inst == nil {
				if _, logged := s.warnings[id]; !logged {
					s.warnings[id] = sc.Name
				}
				continue
			}
			s.insts[id] = inst
		}
		s.ctx.World = s.world
		s.ctx.Entity = id
		s.ctx.Params = sc.Params
		if !s.started[id] {
			s.started[id] = true
			inst.OnStart(&s.ctx)
		}
		inst.OnUpdate(&s.ctx, dt)
	}
}
