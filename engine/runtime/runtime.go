package runtime

import (
	"errors"
	"time"

	"goenginekenga/engine/ecs"
	"goenginekenga/engine/physics"
	"goenginekenga/engine/profiler"
	"goenginekenga/engine/scene"
)

type Mode int

const (
	ModeEdit Mode = iota
	ModePlay
)

type Runtime struct {
	EditWorld        *ecs.World
	PlayWorld        *ecs.World
	PhysicsWorld     *physics.PhysicsWorld
	CollisionManager *physics.CollisionManager
	Constraints      []physics.Constraint // DistanceConstraint, FixedPointConstraint и т.д.
	Profiler         *profiler.Profiler
	Quality          *QualitySystem
	Mode             Mode

	lastTick time.Time
}

func NewFromScene(s *scene.Scene) *Runtime {
	w := s.ToWorld()
	rt := &Runtime{
		EditWorld:        w,
		PhysicsWorld:     physics.DefaultPhysicsWorld(),
		CollisionManager: physics.NewCollisionManager(2.0),
		Profiler:         profiler.NewProfiler(),
		Quality:          NewQualitySystem(),
		Mode:             ModeEdit,
		lastTick:         time.Now(),
	}

	// Настраиваем базовые коллбеки коллизий; конкретная интеграция
	// с удалённым API будет навешана поверх Runtime.
	rt.CollisionManager.OnCollisionEnter = func(a, b physics.EntityID, info *physics.CollisionInfo) {
		_ = a
		_ = b
		_ = info
		// v0: заглушка; события будут проброшены во внешний мир
		// через систему уведомлений WebSocket (см. collision-events).
	}

	return rt
}

// GetProfiler возвращает профилировщик (для overlay FPS, отчётов).
func (rt *Runtime) GetProfiler() *profiler.Profiler {
	return rt.Profiler
}

func (rt *Runtime) StartPlay() {
	if rt.EditWorld == nil {
		return
	}
	rt.PlayWorld = rt.EditWorld.Clone()
	rt.Mode = ModePlay
}

func (rt *Runtime) StopPlay() {
	rt.PlayWorld = nil
	rt.Mode = ModeEdit
}

// ReplaceFromScene заменяет EditWorld новым world из сцены. Для hot-reload при изменении сцены на диске.
func (rt *Runtime) ReplaceFromScene(s *scene.Scene) {
	if s == nil {
		return
	}
	rt.EditWorld = s.ToWorld()
	if rt.Mode == ModePlay {
		rt.PlayWorld = rt.EditWorld.Clone()
	}
}

func (rt *Runtime) ActiveWorld() (*ecs.World, error) {
	switch rt.Mode {
	case ModeEdit:
		if rt.EditWorld == nil {
			return nil, errors.New("no edit world")
		}
		return rt.EditWorld, nil
	case ModePlay:
		if rt.PlayWorld == nil {
			return nil, errors.New("no play world")
		}
		return rt.PlayWorld, nil
	default:
		return nil, errors.New("unknown mode")
	}
}
