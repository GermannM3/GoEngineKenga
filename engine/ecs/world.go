package ecs

import (
	"image/color"
	"sync"

	emath "goenginekenga/engine/math"
	"goenginekenga/engine/physics"
)

type EntityID uint64

type Transform struct {
	Position emath.Vec3 `json:"position"`
	Rotation emath.Vec3 `json:"rotation"` // Euler degrees (v0)
	Scale    emath.Vec3 `json:"scale"`
}

type Camera struct {
	FovYDegrees float32 `json:"fovYDegrees"`
	Near        float32 `json:"near"`
	Far         float32 `json:"far"`
}

type MeshRenderer struct {
	MeshAssetID     string `json:"meshAssetId"`     // UUID as string
	MaterialAssetID string `json:"materialAssetId"` // UUID as string
	ColorR          uint8  `json:"colorR"`
	ColorG          uint8  `json:"colorG"`
	ColorB          uint8  `json:"colorB"`
	ColorA          uint8  `json:"colorA"`
}

type Light struct {
	Kind      string     `json:"kind"` // directional/point/ambient
	ColorRGB  emath.Vec3 `json:"colorRGB"`
	ColorR    uint8      `json:"colorR"`
	ColorG    uint8      `json:"colorG"`
	ColorB    uint8      `json:"colorB"`
	Intensity float32    `json:"intensity"`
	Range     float32    `json:"range"` // for point lights
}

// Rigidbody и Collider определены в пакете physics
type Rigidbody = physics.Rigidbody
type Collider = physics.Collider

type AudioSource struct {
	Clip        string  `json:"clip"`   // asset ID аудиоклипа
	Volume      float32 `json:"volume"` // 0.0 - 1.0
	Pitch       float32 `json:"pitch"`  // 1.0 = normal speed
	Loop        bool    `json:"loop"`
	PlayOnStart bool    `json:"playOnStart"`
	Spatial     bool    `json:"spatial"`     // 3D sound
	MinDistance float32 `json:"minDistance"` // для 3D звука
	MaxDistance float32 `json:"maxDistance"` // для 3D звука
}

type UICanvas struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Trajectory описывает набор 3D-точек для визуализации траектории движения.
type Trajectory struct {
	Points []emath.Vec3 `json:"points"`

	// Параметры отрисовки (v0: используются только цвет и толщина).
	Color color.RGBA `json:"color"`
	Width float32    `json:"width"`
}

// Joint описывает простой сустав робота, привязанный к сущности.
// В v0 это лишь вспомогательные данные поверх Transform.
type Joint struct {
	Name  string      `json:"name"`
	Axis  emath.Vec3 `json:"axis"`
	Angle float32    `json:"angle"`
}

// Dispenser описывает простой «наносчик» мастики, привязанный к сущности.
// В v0 он порождает точки траектории вдоль движения объекта.
type Dispenser struct {
	Active   bool       `json:"active"`
	FlowRate float32    `json:"flowRate"`
	Radius   float32    `json:"radius"`
	Color    color.RGBA `json:"color"`

	LastPosition emath.Vec3 `json:"-"`
	HasLast      bool       `json:"-"`
}

type World struct {
	mu     sync.RWMutex
	nextID EntityID

	order []EntityID

	transforms    map[EntityID]Transform
	cameras       map[EntityID]Camera
	meshRenderers map[EntityID]MeshRenderer
	lights        map[EntityID]Light
	rigidbodies   map[EntityID]Rigidbody
	colliders     map[EntityID]Collider
	audioSources  map[EntityID]AudioSource
	uiCanvases    map[EntityID]UICanvas

	dispensers map[EntityID]Dispenser
	joints map[EntityID]Joint

	trajectories map[EntityID]Trajectory

	names map[EntityID]string
}

func NewWorld() *World {
	return &World{
		nextID:        1,
		order:         nil,
		transforms:    map[EntityID]Transform{},
		cameras:       map[EntityID]Camera{},
		meshRenderers: map[EntityID]MeshRenderer{},
		lights:        map[EntityID]Light{},
		rigidbodies:   map[EntityID]Rigidbody{},
		colliders:     map[EntityID]Collider{},
		audioSources:  map[EntityID]AudioSource{},
		uiCanvases:    map[EntityID]UICanvas{},
		dispensers:    map[EntityID]Dispenser{},
		joints:        map[EntityID]Joint{},
		trajectories:  map[EntityID]Trajectory{},
		names:         map[EntityID]string{},
	}
}

func (w *World) CreateEntity(name string) EntityID {
	w.mu.Lock()
	defer w.mu.Unlock()
	id := w.nextID
	w.nextID++
	w.order = append(w.order, id)
	if name != "" {
		w.names[id] = name
	}
	return id
}

func (w *World) Entities() []EntityID {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]EntityID, len(w.order))
	copy(out, w.order)
	return out
}

func (w *World) Name(id EntityID) string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.names[id]
}

func (w *World) SetName(id EntityID, name string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if name == "" {
		delete(w.names, id)
		return
	}
	w.names[id] = name
}

func (w *World) SetTransform(id EntityID, t Transform) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.transforms[id] = t
}

func (w *World) GetTransform(id EntityID) (Transform, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	t, ok := w.transforms[id]
	return t, ok
}

func (w *World) SetCamera(id EntityID, c Camera) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.cameras[id] = c
}

func (w *World) GetCamera(id EntityID) (Camera, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	c, ok := w.cameras[id]
	return c, ok
}

func (w *World) SetMeshRenderer(id EntityID, mr MeshRenderer) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.meshRenderers[id] = mr
}

func (w *World) GetMeshRenderer(id EntityID) (MeshRenderer, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	mr, ok := w.meshRenderers[id]
	return mr, ok
}

func (w *World) SetLight(id EntityID, l Light) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lights[id] = l
}

func (w *World) GetLight(id EntityID) (Light, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	l, ok := w.lights[id]
	return l, ok
}

func (w *World) SetRigidbody(id EntityID, rb Rigidbody) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.rigidbodies[id] = rb
}

func (w *World) GetRigidbody(id EntityID) (Rigidbody, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	rb, ok := w.rigidbodies[id]
	return rb, ok
}

func (w *World) SetCollider(id EntityID, c Collider) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.colliders[id] = c
}

func (w *World) GetCollider(id EntityID) (Collider, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	c, ok := w.colliders[id]
	return c, ok
}

func (w *World) SetAudioSource(id EntityID, as AudioSource) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.audioSources[id] = as
}

func (w *World) GetAudioSource(id EntityID) (AudioSource, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	as, ok := w.audioSources[id]
	return as, ok
}

func (w *World) SetUICanvas(id EntityID, canvas UICanvas) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.uiCanvases[id] = canvas
}

func (w *World) GetUICanvas(id EntityID) (UICanvas, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	canvas, ok := w.uiCanvases[id]
	return canvas, ok
}

// SetTrajectory задаёт или обновляет траекторию для сущности.
func (w *World) SetTrajectory(id EntityID, t Trajectory) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.trajectories[id] = t
}

// GetTrajectory возвращает траекторию сущности, если она есть.
func (w *World) GetTrajectory(id EntityID) (Trajectory, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	t, ok := w.trajectories[id]
	return t, ok
}

// SetJoint задаёт параметры сустава для сущности.
func (w *World) SetJoint(id EntityID, j Joint) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.joints[id] = j
}

// GetJoint возвращает сустав сущности, если он задан.
func (w *World) GetJoint(id EntityID) (Joint, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	j, ok := w.joints[id]
	return j, ok
}

// SetDispenser задаёт или обновляет параметры нанесения мастики.
func (w *World) SetDispenser(id EntityID, d Dispenser) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.dispensers[id] = d
}

// GetDispenser возвращает Dispenser сущности, если он есть.
func (w *World) GetDispenser(id EntityID) (Dispenser, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	d, ok := w.dispensers[id]
	return d, ok
}

// Clone делает глубокую копию мира для PlayMode.
func (w *World) Clone() *World {
	w.mu.RLock()
	defer w.mu.RUnlock()
	nw := NewWorld()
	nw.nextID = w.nextID
	nw.order = append(nw.order, w.order...)
	for k, v := range w.transforms {
		nw.transforms[k] = v
	}
	for k, v := range w.cameras {
		nw.cameras[k] = v
	}
	for k, v := range w.meshRenderers {
		nw.meshRenderers[k] = v
	}
	for k, v := range w.lights {
		nw.lights[k] = v
	}
	for k, v := range w.rigidbodies {
		nw.rigidbodies[k] = v
	}
	for k, v := range w.colliders {
		nw.colliders[k] = v
	}
	for k, v := range w.audioSources {
		nw.audioSources[k] = v
	}
	for k, v := range w.uiCanvases {
		nw.uiCanvases[k] = v
	}
	for k, v := range w.dispensers {
		nw.dispensers[k] = v
	}
	for k, v := range w.joints {
		nw.joints[k] = v
	}
	for k, v := range w.trajectories {
		nw.trajectories[k] = v
	}
	for k, v := range w.names {
		nw.names[k] = v
	}
	return nw
}
