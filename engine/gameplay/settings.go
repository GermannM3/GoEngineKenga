package gameplay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Settings — настройки игры (инди: громкость, полноэкран, качество).
type Settings struct {
	mu sync.RWMutex

	// Аудио
	MasterVolume float32 `json:"masterVolume"` // 0..1
	MusicVolume  float32 `json:"musicVolume"`
	SFXVolume    float32 `json:"sfxVolume"`

	// Видео
	Fullscreen bool `json:"fullscreen"`
	VSync      bool `json:"vsync"`
	Width      int  `json:"width"`
	Height     int  `json:"height"`

	// Качество (используется QualitySystem)
	QualityPreset string `json:"qualityPreset"` // low, medium, high, ultra

	// Гейм play
	Language         string  `json:"language"`
	MouseSensitivity float32 `json:"mouseSensitivity"`
}

// DefaultSettings возвращает настройки по умолчанию.
func DefaultSettings() *Settings {
	return &Settings{
		MasterVolume:     1.0,
		MusicVolume:      0.8,
		SFXVolume:        1.0,
		Fullscreen:       false,
		VSync:            true,
		Width:            1280,
		Height:           720,
		QualityPreset:    "medium",
		Language:         "en",
		MouseSensitivity: 1.0,
	}
}

// Load загружает настройки из файла.
func (s *Settings) Load(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, s)
}

// Save сохраняет настройки в файл.
func (s *Settings) Save(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}

// GetMasterVolume возвращает громкость мастера.
func (s *Settings) GetMasterVolume() float32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.MasterVolume
}

// SetMasterVolume устанавливает громкость мастера.
func (s *Settings) SetMasterVolume(v float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	s.MasterVolume = v
}

// GetQualityPreset возвращает пресет качества.
func (s *Settings) GetQualityPreset() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.QualityPreset
}

// SetQualityPreset устанавливает пресет (low, medium, high, ultra).
func (s *Settings) SetQualityPreset(preset string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.QualityPreset = preset
}
