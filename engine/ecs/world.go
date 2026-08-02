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

// Health — здоровье сущности (как в Unity). Current <= 0 считается гибелью.
type Health struct {
	Current float32 `json:"current"`
	Max     float32 `json:"max"`
}

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

// SpriteRenderer компонент для 2D-спрайтов
type SpriteRenderer struct {
	TexturePath string `json:"texturePath"` // путь к PNG относительно assets/
	Layer       int    `json:"layer"`       // порядок отрисовки (0=фон, 10=враги, 20=герой, 100=UI)
	FlipX       bool   `json:"flipX"`
	FlipY       bool   `json:"flipY"`
	Visible     bool   `json:"visible"`
	// Source rect для спрайт-листов (item_set.png)
	SrcX int `json:"srcX,omitempty"` // 0 = весь спрайт
	SrcY int `json:"srcY,omitempty"`
	SrcW int `json:"srcW,omitempty"`
	SrcH int `json:"srcH,omitempty"`
	// Tint color
	ColorR uint8 `json:"colorR,omitempty"`
	ColorG uint8 `json:"colorG,omitempty"`
	ColorB uint8 `json:"colorB,omitempty"`
	ColorA uint8 `json:"colorA,omitempty"`
}

// Camera2D компонент для 2D-камеры
type Camera2D struct {
	Zoom     float32  `json:"zoom"`      // 1.0 = 100%
	FollowID EntityID `json:"followId"`  // entity to follow (0 = static)
}

// AnimationState определяет текущее состояние анимации
type AnimationState struct {
	CurrentClip     string  `json:"currentClip"`     // название текущей анимации
	CurrentFrame    int     `json:"currentFrame"`    // текущий кадр
	FrameTime       float32 `json:"frameTime"`       // время на кадр
	ElapsedTime     float32 `json:"elapsedTime"`     // прошедшее время с начала анимации
	IsPlaying       bool    `json:"isPlaying"`       // воспроизводится ли анимация
	Loop            bool    `json:"loop"`            // зациклена ли анимация
	Speed           float32 `json:"speed"`           // скорость воспроизведения
}

// AnimationClip определяет клип анимации
type AnimationClip struct {
	Name           string  `json:"name"`           // название анимации
	Frames         []int   `json:"frames"`         // индексы кадров в спрайт-листе
	Duration       float32 `json:"duration"`       // длительность в секундах
	FrameDuration  float32 `json:"frameDuration"`  // длительность одного кадра
	Loop           bool    `json:"loop"`           // зациклена ли анимация
	Layout         string  `json:"layout"`         // способ организации кадров: "horizontal", "vertical", "grid"
	StartX         int     `json:"startX"`         // начальная позиция X для первого кадра
	StartY         int     `json:"startY"`         // начальная позиция Y для первого кадра
	StepX          int     `json:"stepX"`         // шаг по X между кадрами
	StepY          int     `json:"stepY"`         // шаг по Y между кадрами
	FrameWidth     int     `json:"frameWidth"`     // ширина одного кадра
	FrameHeight    int     `json:"frameHeight"`    // высота одного кадра
}

// AnimationController управляет анимациями спрайта
type AnimationController struct {
	Clips          []AnimationClip `json:"clips"`           // доступные анимации
	DefaultClip    string          `json:"defaultClip"`     // анимация по умолчанию
	CurrentClip    string          `json:"currentClip"`     // текущая анимация
	PlaybackSpeed  float32         `json:"playbackSpeed"`   // общая скорость воспроизведения
	AutoPlay       bool            `json:"autoPlay"`        // автовоспроизведение
}

// Animator — компонент 3D skeletal-анимации (glTF skins).
// SkeletonPath — путь к .skeleton.json, ClipPaths — пути к .clip.json (относительно проекта).
// Клипы импортируются из glTF (Animation), скелет — из Skin.
type Animator struct {
	SkeletonPath string   `json:"skeletonPath,omitempty"` // путь к .skeleton.json
	ClipPaths    []string `json:"clipPaths,omitempty"`    // пути к .clip.json
	Clip         string   `json:"clip,omitempty"`         // имя текущего клипа
	Speed        float32  `json:"speed"`                  // 1.0 = нормальная скорость
	Loop         bool     `json:"loop"`
	AutoPlay     bool     `json:"autoPlay"`

	// Runtime (не сериализуется): финальные bone matrices для GPU (16 floats на кость,
	// column-major, до 64 костей). Заполняет SkeletalAnimationSystem.
	BoneMatrices []float32 `json:"-"`
}

// Kart — аркадный картинг: машина (Atom, Moskvich M70, M90), лапы, бонус.
// CarID: "atom" | "moskvich_m70" | "moskvich_m90"
// Скорость и позиция — в Rigidbody и Transform.
type Kart struct {
	CarID         string  `json:"carId"`
	CurrentLap    int     `json:"currentLap"`
	LastCheckpoint int    `json:"lastCheckpoint"` // индекс чекпоинта для детекции круга
	PowerUp       string  `json:"powerUp"`        // nitro, shield, rocket, oil, electric_boost, ""
	PowerUpTimer  float32 `json:"powerUpTimer"`   // время действия бонуса
	IsBot         bool    `json:"isBot"`
	RacePosition  int     `json:"racePosition"` // 1-based на основе прогресса

	// Тайминги кругов (секунды, относительно RaceState.RaceTime)
	LapStartTime float32 `json:"lapStartTime"` // время старта текущего круга
	LastLapTime  float32 `json:"lastLapTime"`  // время последнего круга, с (0 = ещё нет)
	BestLapTime  float32 `json:"bestLapTime"`  // лучший круг, с (0 = ещё нет)
}

// PowerUpPickup — коробка/иконка бонуса на трассе (триггер).
// Type: nitro, shield, rocket, oil, electric_boost
type PowerUpPickup struct {
	Type       string  `json:"type"`
	RespawnSec float32 `json:"respawnSec"` // 0 = не респавнится
	Cooldown   float32 `json:"-"`         // оставшееся время до появления
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
	spriteRenderers map[EntityID]SpriteRenderer
	camera2Ds     map[EntityID]Camera2D
	lights        map[EntityID]Light
	rigidbodies   map[EntityID]Rigidbody
	colliders     map[EntityID]Collider
	healths       map[EntityID]Health
	audioSources  map[EntityID]AudioSource
	uiCanvases    map[EntityID]UICanvas

	dispensers map[EntityID]Dispenser
	joints map[EntityID]Joint

	trajectories map[EntityID]Trajectory

	animationStates map[EntityID]AnimationState
	animationControllers map[EntityID]AnimationController
	animators     map[EntityID]Animator

	karts        map[EntityID]Kart
	powerUpPickups map[EntityID]PowerUpPickup

	names map[EntityID]string
}

func NewWorld() *World {
	return &World{
		nextID:        1,
		order:         nil,
		transforms:    map[EntityID]Transform{},
		cameras:       map[EntityID]Camera{},
		meshRenderers: map[EntityID]MeshRenderer{},
		spriteRenderers: map[EntityID]SpriteRenderer{},
		camera2Ds:     map[EntityID]Camera2D{},
		lights:        map[EntityID]Light{},
		rigidbodies:   map[EntityID]Rigidbody{},
		colliders:     map[EntityID]Collider{},
		healths:       map[EntityID]Health{},
		audioSources:  map[EntityID]AudioSource{},
		uiCanvases:    map[EntityID]UICanvas{},
		dispensers:    map[EntityID]Dispenser{},
		joints:        map[EntityID]Joint{},
		trajectories:  map[EntityID]Trajectory{},
		animationStates: map[EntityID]AnimationState{},
		animationControllers: map[EntityID]AnimationController{},
		animators:     map[EntityID]Animator{},
		karts:         map[EntityID]Kart{},
		powerUpPickups: map[EntityID]PowerUpPickup{},
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

func (w *World) SetHealth(id EntityID, h Health) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.healths[id] = h
}

func (w *World) GetHealth(id EntityID) (Health, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	h, ok := w.healths[id]
	return h, ok
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

func (w *World) SetSpriteRenderer(id EntityID, sr SpriteRenderer) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.spriteRenderers[id] = sr
}

func (w *World) GetSpriteRenderer(id EntityID) (SpriteRenderer, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	sr, ok := w.spriteRenderers[id]
	return sr, ok
}

func (w *World) SetCamera2D(id EntityID, cam2D Camera2D) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.camera2Ds[id] = cam2D
}

func (w *World) GetCamera2D(id EntityID) (Camera2D, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	cam2D, ok := w.camera2Ds[id]
	return cam2D, ok
}

func (w *World) SetKart(id EntityID, k Kart) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.karts[id] = k
}

func (w *World) GetKart(id EntityID) (Kart, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	k, ok := w.karts[id]
	return k, ok
}

func (w *World) SetPowerUpPickup(id EntityID, p PowerUpPickup) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.powerUpPickups[id] = p
}

func (w *World) GetPowerUpPickup(id EntityID) (PowerUpPickup, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	p, ok := w.powerUpPickups[id]
	return p, ok
}

func (w *World) SetAnimationState(id EntityID, animState AnimationState) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.animationStates[id] = animState
}

func (w *World) GetAnimationState(id EntityID) (AnimationState, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	animState, ok := w.animationStates[id]
	return animState, ok
}

func (w *World) SetAnimationController(id EntityID, animCtrl AnimationController) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.animationControllers[id] = animCtrl
}

func (w *World) GetAnimationController(id EntityID) (AnimationController, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	animCtrl, ok := w.animationControllers[id]
	return animCtrl, ok
}

func (w *World) SetAnimator(id EntityID, a Animator) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.animators[id] = a
}

func (w *World) GetAnimator(id EntityID) (Animator, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	a, ok := w.animators[id]
	return a, ok
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
	for k, v := range w.spriteRenderers {
		nw.spriteRenderers[k] = v
	}
	for k, v := range w.camera2Ds {
		nw.camera2Ds[k] = v
	}
	for k, v := range w.animationStates {
		nw.animationStates[k] = v
	}
	for k, v := range w.animationControllers {
		nw.animationControllers[k] = v
	}
	for k, v := range w.animators {
		if v.BoneMatrices != nil {
			v.BoneMatrices = append([]float32(nil), v.BoneMatrices...)
		}
		nw.animators[k] = v
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
	for k, v := range w.healths {
		nw.healths[k] = v
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
	for k, v := range w.karts {
		nw.karts[k] = v
	}
	for k, v := range w.powerUpPickups {
		nw.powerUpPickups[k] = v
	}
	for k, v := range w.names {
		nw.names[k] = v
	}
	return nw
}
