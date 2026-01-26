package runtime

import "sync"

// QualityPreset — пресет качества (AAA-ориентировано: low/medium/high/ultra).
type QualityPreset string

const (
	QualityLow    QualityPreset = "low"
	QualityMedium QualityPreset = "medium"
	QualityHigh   QualityPreset = "high"
	QualityUltra  QualityPreset = "ultra"
)

// QualityConfig — конфиг качества (LOD, партиклы, тени, постпроцесс и т.д.).
type QualityConfig struct {
	Preset QualityPreset

	// Рендер
	MaxDrawCalls     int  `json:"maxDrawCalls"`
	MaxTriangles     int  `json:"maxTriangles"`
	ShadowResolution int  `json:"shadowResolution"` // 0=off, 512, 1024, 2048
	PostProcess      bool `json:"postProcess"`
	ParticleLimit    int  `json:"particleLimit"`

	// Физика
	PhysicsSubsteps int `json:"physicsSubsteps"`

	// Аудио
	MaxVoices  int `json:"maxVoices"`
	SampleRate int `json:"sampleRate"`

	// Прочее
	LODBias float32 `json:"lodBias"` // 0..2, выше = детальнее
}

// QualitySystem управляет пресетами качества.
type QualitySystem struct {
	mu      sync.RWMutex
	presets map[QualityPreset]*QualityConfig
	current QualityPreset
}

// NewQualitySystem создаёт систему качества с дефолтными пресетами.
func NewQualitySystem() *QualitySystem {
	qs := &QualitySystem{
		presets: make(map[QualityPreset]*QualityConfig),
		current: QualityMedium,
	}

	qs.presets[QualityLow] = &QualityConfig{
		Preset:           QualityLow,
		MaxDrawCalls:     512,
		MaxTriangles:     50_000,
		ShadowResolution: 0,
		PostProcess:      false,
		ParticleLimit:    500,
		PhysicsSubsteps:  1,
		MaxVoices:        16,
		SampleRate:       22050,
		LODBias:          0.5,
	}

	qs.presets[QualityMedium] = &QualityConfig{
		Preset:           QualityMedium,
		MaxDrawCalls:     2048,
		MaxTriangles:     200_000,
		ShadowResolution: 512,
		PostProcess:      true,
		ParticleLimit:    2000,
		PhysicsSubsteps:  2,
		MaxVoices:        32,
		SampleRate:       44100,
		LODBias:          1.0,
	}

	qs.presets[QualityHigh] = &QualityConfig{
		Preset:           QualityHigh,
		MaxDrawCalls:     8192,
		MaxTriangles:     1_000_000,
		ShadowResolution: 1024,
		PostProcess:      true,
		ParticleLimit:    5000,
		PhysicsSubsteps:  3,
		MaxVoices:        64,
		SampleRate:       48000,
		LODBias:          1.25,
	}

	qs.presets[QualityUltra] = &QualityConfig{
		Preset:           QualityUltra,
		MaxDrawCalls:     16384,
		MaxTriangles:     5_000_000,
		ShadowResolution: 2048,
		PostProcess:      true,
		ParticleLimit:    10000,
		PhysicsSubsteps:  4,
		MaxVoices:        128,
		SampleRate:       48000,
		LODBias:          1.5,
	}

	return qs
}

// SetPreset переключает пресет.
func (qs *QualitySystem) SetPreset(p QualityPreset) {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	if _, ok := qs.presets[p]; ok {
		qs.current = p
	}
}

// GetConfig возвращает текущий конфиг качества.
func (qs *QualitySystem) GetConfig() *QualityConfig {
	qs.mu.RLock()
	defer qs.mu.RUnlock()
	c := qs.presets[qs.current]
	if c == nil {
		return qs.presets[QualityMedium]
	}
	return c
}

// GetPreset возвращает текущий пресет.
func (qs *QualitySystem) GetPreset() QualityPreset {
	qs.mu.RLock()
	defer qs.mu.RUnlock()
	return qs.current
}
