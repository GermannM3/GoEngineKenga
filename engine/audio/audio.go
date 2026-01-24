package audio

import (
	"io"
	"time"

	emath "goenginekenga/engine/math"
)

// AudioClip представляет загруженный аудиоклип
type AudioClip struct {
	Name   string
	Data   []byte
	Format string // wav, mp3, ogg
}

// AudioSource представляет источник звука в 3D пространстве
type AudioSource struct {
	Clip        string      `json:"clip"`        // asset ID аудиоклипа
	Volume      float32     `json:"volume"`      // 0.0 - 1.0
	Pitch       float32     `json:"pitch"`       // 1.0 = normal speed
	Loop        bool        `json:"loop"`
	PlayOnStart bool        `json:"playOnStart"`
	Spatial     bool        `json:"spatial"`     // 3D sound
	MinDistance float32     `json:"minDistance"` // для 3D звука
	MaxDistance float32     `json:"maxDistance"` // для 3D звука
	Position    emath.Vec3  `json:"position"`    // позиция источника

	// Runtime fields
	isPlaying bool
}

// DefaultAudioSource возвращает стандартный audio source
func DefaultAudioSource() *AudioSource {
	return &AudioSource{
		Volume:      1.0,
		Pitch:       1.0,
		Loop:        false,
		PlayOnStart: false,
		Spatial:     false,
		MinDistance: 1.0,
		MaxDistance: 50.0,
		Position:    emath.V3(0, 0, 0),
		isPlaying:   false,
	}
}

// AudioListener представляет слушателя звука (обычно камеру)
type AudioListener struct {
	Position emath.Vec3 `json:"position"`
	// В будущем можно добавить rotation для directional audio
}

// AudioEngine управляет аудиосистемой
type AudioEngine struct {
	sources   map[string]*AudioSource // key: entity ID
	listener  *AudioListener
	masterVolume float32
}

// NewAudioEngine создает новый audio engine
func NewAudioEngine() *AudioEngine {
	return &AudioEngine{
		sources:      make(map[string]*AudioSource),
		listener:     &AudioListener{Position: emath.V3(0, 0, 0)},
		masterVolume: 1.0,
	}
}

// SetListener устанавливает позицию слушателя
func (ae *AudioEngine) SetListener(position emath.Vec3) {
	ae.listener.Position = position
}

// AddSource добавляет audio source
func (ae *AudioEngine) AddSource(entityID string, source *AudioSource) {
	ae.sources[entityID] = source
}

// RemoveSource удаляет audio source
func (ae *AudioEngine) RemoveSource(entityID string) {
	delete(ae.sources, entityID)
}

// Play начинает воспроизведение
func (ae *AudioEngine) Play(entityID string) {
	if source, ok := ae.sources[entityID]; ok {
		source.isPlaying = true
		// TODO: Реализовать фактическое воспроизведение через audio backend
	}
}

// Stop останавливает воспроизведение
func (ae *AudioEngine) Stop(entityID string) {
	if source, ok := ae.sources[entityID]; ok {
		source.isPlaying = false
		// TODO: Реализовать фактическое останов воспроизведения
	}
}

// Update обновляет состояние аудио (вызывается каждый кадр)
func (ae *AudioEngine) Update(deltaTime time.Duration) {
	// Обновляем 3D позиционирование
	for _, source := range ae.sources {
		if source.Spatial {
			distance := ae.calculateDistance(source.Position, ae.listener.Position)
			volume := ae.calculateSpatialVolume(distance, source.MinDistance, source.MaxDistance)
			source.Volume = volume * ae.masterVolume
		}
	}
}

// calculateDistance вычисляет расстояние между двумя точками
func (ae *AudioEngine) calculateDistance(a, b emath.Vec3) float32 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return sqrt(dx*dx + dy*dy + dz*dz)
}

// calculateSpatialVolume вычисляет громкость для 3D звука
func (ae *AudioEngine) calculateSpatialVolume(distance, minDist, maxDist float32) float32 {
	if distance <= minDist {
		return 1.0
	}
	if distance >= maxDist {
		return 0.0
	}
	// Линейное затухание
	return 1.0 - (distance-minDist)/(maxDist-minDist)
}

// sqrt простая реализация квадратного корня (для примера)
func sqrt(x float32) float32 {
	if x <= 0 {
		return 0
	}
	// Быстрый метод Ньютона
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) * 0.5
	}
	return z
}

// SetMasterVolume устанавливает общую громкость
func (ae *AudioEngine) SetMasterVolume(volume float32) {
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}
	ae.masterVolume = volume
}