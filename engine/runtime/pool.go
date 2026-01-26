package runtime

import (
	"sync"
)

// ObjectPool — пул объектов для переиспользования (AAA: меньше аллокаций, стабильный FPS).
type ObjectPool struct {
	mu    sync.Mutex
	New   func() interface{}
	Reset func(interface{})
	items []interface{}
	inUse int
}

// NewObjectPool создаёт пул с фабрикой и сбросом.
func NewObjectPool(newFn func() interface{}, resetFn func(interface{}), cap int) *ObjectPool {
	if cap <= 0 {
		cap = 64
	}
	return &ObjectPool{
		New:   newFn,
		Reset: resetFn,
		items: make([]interface{}, 0, cap),
	}
}

// Get выдаёт объект из пула или создаёт новый.
func (p *ObjectPool) Get() interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	var x interface{}
	if len(p.items) > 0 {
		x = p.items[len(p.items)-1]
		p.items = p.items[:len(p.items)-1]
		if p.Reset != nil {
			p.Reset(x)
		}
	} else if p.New != nil {
		x = p.New()
	}
	if x != nil {
		p.inUse++
	}
	return x
}

// Put возвращает объект в пул.
func (p *ObjectPool) Put(x interface{}) {
	if x == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Reset != nil {
		p.Reset(x)
	}
	p.items = append(p.items, x)
	if p.inUse > 0 {
		p.inUse--
	}
}

// InUse возвращает число выданных объектов.
func (p *ObjectPool) InUse() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.inUse
}

// Len возвращает размер пула (доступных объектов).
func (p *ObjectPool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.items)
}
