package runtime

import (
	"errors"
	"time"

	"goenginekenga/engine/ecs"
	"goenginekenga/engine/physics"
	"goenginekenga/engine/scene"
)

type Mode int

const (
	ModeEdit Mode = iota
	ModePlay
)

type Runtime struct {
	EditWorld   *ecs.World
	PlayWorld   *ecs.World
	PhysicsWorld *physics.PhysicsWorld
	Mode        Mode

	lastTick time.Time
}

func NewFromScene(s *scene.Scene) *Runtime {
	w := s.ToWorld()
	return &Runtime{
		EditWorld:   w,
		PhysicsWorld: physics.DefaultPhysicsWorld(),
		Mode:        ModeEdit,
		lastTick:    time.Now(),
	}
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

