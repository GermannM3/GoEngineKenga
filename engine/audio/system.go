package audio

import (
	"fmt"
	"time"

	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
)

// ClipLoader загружает аудиоклип по asset ID (реализует *asset.Resolver —
// интерфейс, чтобы не создавать цикл импортов asset <-> audio).
type ClipLoader interface {
	ResolveAudioClip(assetID string) (*AudioClip, error)
}

// AudioSystem связывает ECS-компоненты AudioSource с AudioEngine:
// подгружает клипы по asset ID, синхронизирует позиции источников,
// проигрывает PlayOnStart и обновляет spatial-громкость по позиции камеры.
type AudioSystem struct {
	engine     *AudioEngine
	loader     ClipLoader
	loaded     map[string]bool
	failed     map[string]bool
	started    map[ecs.EntityID]bool
	sourceKeys map[ecs.EntityID]string
}

// NewAudioSystem создаёт аудиосистему. loader == nil — только один-шоты
// без подгрузки клипов (например, когда резолвер недоступен).
func NewAudioSystem(loader ClipLoader) *AudioSystem {
	return &AudioSystem{
		engine:     NewAudioEngine(),
		loader:     loader,
		loaded:     map[string]bool{},
		failed:     map[string]bool{},
		started:    map[ecs.EntityID]bool{},
		sourceKeys: map[ecs.EntityID]string{},
	}
}

// Engine возвращает AudioEngine (для PlayOneShot без ECS).
func (as *AudioSystem) Engine() *AudioEngine { return as.engine }

// EnsureClip подгружает клип в движок, если ещё не загружен.
func (as *AudioSystem) EnsureClip(assetID string) bool {
	if assetID == "" {
		return false
	}
	if as.loaded[assetID] {
		return true
	}
	if as.failed[assetID] {
		return false
	}
	if as.loader == nil {
		as.failed[assetID] = true
		return false
	}
	clip, err := as.loader.ResolveAudioClip(assetID)
	if err != nil {
		as.failed[assetID] = true
		return false
	}
	as.engine.RegisterClip(assetID, clip)
	as.loaded[assetID] = true
	return true
}

// PlayOneShot проигрывает клип один раз (по asset ID) в мировой позиции.
func (as *AudioSystem) PlayOneShot(clipAssetID string, position emath.Vec3, volume float32) {
	if !as.EnsureClip(clipAssetID) {
		return
	}
	as.engine.PlayOneShot(clipAssetID, position, volume)
}

// Update вызывается каждый кадр: синхронизирует источники из World.
func (as *AudioSystem) Update(w *ecs.World, dt time.Duration) {
	if w == nil {
		return
	}

	// Слушатель — первая камера
	for _, id := range w.Entities() {
		if _, ok := w.GetCamera(id); ok {
			if tr, ok := w.GetTransform(id); ok {
				as.engine.SetListener(tr.Position)
			}
			break
		}
	}

	// Актуальные источники
	seen := map[ecs.EntityID]bool{}
	for _, id := range w.Entities() {
		src, ok := w.GetAudioSource(id)
		if !ok {
			continue
		}
		seen[id] = true

		key, hasKey := as.sourceKeys[id]
		if !hasKey {
			key = fmt.Sprintf("entity_%d", id)
			as.sourceKeys[id] = key
		}

		// Позиция источника — из трансформа
		srcPos := emath.V3(0, 0, 0)
		if tr, ok := w.GetTransform(id); ok {
			srcPos = tr.Position
		}

		if !as.EnsureClip(src.Clip) {
			continue
		}

		engSrc := &AudioSource{
			Clip:        src.Clip,
			Volume:      src.Volume,
			Pitch:       src.Pitch,
			Loop:        src.Loop,
			PlayOnStart: src.PlayOnStart,
			Spatial:     src.Spatial,
			MinDistance: src.MinDistance,
			MaxDistance: src.MaxDistance,
			Position:    srcPos,
		}
		as.engine.AddSource(key, engSrc)

		if src.PlayOnStart && !as.started[id] {
			as.engine.Play(key)
			as.started[id] = true
		}
	}

	// Удаляем источники исчезнувших сущностей
	for id, key := range as.sourceKeys {
		if !seen[id] {
			as.engine.Stop(key)
			delete(as.sourceKeys, id)
			delete(as.started, id)
		}
	}

	as.engine.Update(dt)
}
