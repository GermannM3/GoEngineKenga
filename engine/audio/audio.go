package audio

import (
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
	Clip        string     `json:"clip"`   // asset ID аудиоклипа
	Volume      float32    `json:"volume"` // 0.0 - 1.0
	Pitch       float32    `json:"pitch"`  // 1.0 = normal speed
	Loop        bool       `json:"loop"`
	PlayOnStart bool       `json:"playOnStart"`
	Spatial     bool       `json:"spatial"`     // 3D sound
	MinDistance float32    `json:"minDistance"` // для 3D звука
	MaxDistance float32    `json:"maxDistance"` // для 3D звука
	Position    emath.Vec3 `json:"position"`    // позиция источника

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
	sources      map[string]*AudioSource // key: entity ID
	clips        map[string]*AudioClip   // key: clip asset ID
	playerIDs    map[string]string       // entity ID -> player ID
	listener     *AudioListener
	masterVolume float32
	backend      *EbitenAudioBackend
}

// NewAudioEngine создает новый audio engine
func NewAudioEngine() *AudioEngine {
	return &AudioEngine{
		sources:      make(map[string]*AudioSource),
		clips:        make(map[string]*AudioClip),
		playerIDs:    make(map[string]string),
		listener:     &AudioListener{Position: emath.V3(0, 0, 0)},
		masterVolume: 1.0,
		backend:      NewEbitenAudioBackend(),
	}
}

// RegisterClip registers an audio clip for later playback
func (ae *AudioEngine) RegisterClip(assetID string, clip *AudioClip) {
	ae.clips[assetID] = clip
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
	source, ok := ae.sources[entityID]
	if !ok {
		return
	}

	// Get the clip
	clip, ok := ae.clips[source.Clip]
	if !ok || clip == nil {
		return
	}

	// Stop existing playback if any
	if existingPlayerID, ok := ae.playerIDs[entityID]; ok {
		ae.backend.Stop(existingPlayerID)
	}

	// Calculate volume with spatial attenuation
	volume := float64(source.Volume * ae.masterVolume)
	if source.Spatial {
		distance := ae.calculateDistance(source.Position, ae.listener.Position)
		spatialVol := ae.calculateSpatialVolume(distance, source.MinDistance, source.MaxDistance)
		volume *= float64(spatialVol)
	}

	// Play through backend
	playerID, err := ae.backend.PlayFromClip(clip, volume, source.Loop)
	if err == nil {
		ae.playerIDs[entityID] = playerID
		source.isPlaying = true
	}
}

// Stop останавливает воспроизведение
func (ae *AudioEngine) Stop(entityID string) {
	if source, ok := ae.sources[entityID]; ok {
		source.isPlaying = false
	}

	if playerID, ok := ae.playerIDs[entityID]; ok {
		ae.backend.Stop(playerID)
		delete(ae.playerIDs, entityID)
	}
}

// Update обновляет состояние аудио (вызывается каждый кадр)
func (ae *AudioEngine) Update(deltaTime time.Duration) {
	// Update backend (handles looping)
	ae.backend.Update()

	// Обновляем 3D позиционирование
	for entityID, source := range ae.sources {
		if source.Spatial && source.isPlaying {
			distance := ae.calculateDistance(source.Position, ae.listener.Position)
			spatialVol := ae.calculateSpatialVolume(distance, source.MinDistance, source.MaxDistance)
			volume := float64(source.Volume * ae.masterVolume * spatialVol)

			// Update volume in backend
			if playerID, ok := ae.playerIDs[entityID]; ok {
				ae.backend.SetVolume(playerID, volume)
			}
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
