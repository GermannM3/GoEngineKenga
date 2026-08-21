// Package cyberninja — Go-скрипты игры CyberNinja для engine/scripts.
//
// Паттерн для своих игр: пакет со скриптами регистрирует их в init()
// через scripts.Register и импортируется в main вашего exe (или в
// cmd/kenga для демо-контента). Сцена ссылается на скрипт по имени:
//
//	"script": { "name": "spinner", "params": { "speed": "120" } }
package cyberninja

import (
	"math"

	"goenginekenga/engine/scripts"
)

func init() {
	scripts.Register("spinner", func() scripts.Script { return &Spinner{} })
	scripts.Register("patrol_x", func() scripts.Script { return &PatrolX{} })
	scripts.Register("bob", func() scripts.Script { return &Bob{} })
}

// Spinner вращает объект вокруг вертикальной оси.
// Параметры: speed (град/с, по умолчанию 90).
type Spinner struct{}

func (s *Spinner) OnStart(c *scripts.Context) {}

func (s *Spinner) OnUpdate(c *scripts.Context, dt float32) {
	tr, ok := c.World.GetTransform(c.Entity)
	if !ok {
		return
	}
	tr.Rotation.Y += c.Float("speed", 90) * dt
	c.World.SetTransform(c.Entity, tr)
}

// PatrolX патрулирует по оси X вокруг стартовой позиции.
// Параметры: span (полуширина, 5), speed (ед/с, 2).
type PatrolX struct {
	home float32
	dir  float32
}

func (s *PatrolX) OnStart(c *scripts.Context) {
	if tr, ok := c.World.GetTransform(c.Entity); ok {
		s.home = tr.Position.X
	}
	s.dir = 1
}

func (s *PatrolX) OnUpdate(c *scripts.Context, dt float32) {
	tr, ok := c.World.GetTransform(c.Entity)
	if !ok {
		return
	}
	span := c.Float("span", 5)
	speed := c.Float("speed", 2)
	tr.Position.X += s.dir * speed * dt
	if tr.Position.X > s.home+span {
		s.dir = -1
	} else if tr.Position.X < s.home-span {
		s.dir = 1
	}
	c.World.SetTransform(c.Entity, tr)
}

// Bob плавно покачивает объект по вертикали (синусоида).
// Параметры: amplitude (0.3), speed (2).
type Bob struct {
	t float32
	y float32
}

func (s *Bob) OnStart(c *scripts.Context) {
	if tr, ok := c.World.GetTransform(c.Entity); ok {
		s.y = tr.Position.Y
	}
}

func (s *Bob) OnUpdate(c *scripts.Context, dt float32) {
	tr, ok := c.World.GetTransform(c.Entity)
	if !ok {
		return
	}
	s.t += dt * c.Float("speed", 2)
	tr.Position.Y = s.y + float32(math.Sin(float64(s.t)))*c.Float("amplitude", 0.3)
	c.World.SetTransform(c.Entity, tr)
}
