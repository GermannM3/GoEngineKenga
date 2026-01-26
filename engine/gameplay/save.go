package gameplay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SaveSystem управляет сохранением и загрузкой игрового состояния (инди-ориентировано).
type SaveSystem struct {
	mu           sync.RWMutex
	saveDir      string
	slotPrefix   string
	maxSlots     int
	autoSave     bool
	autoInterval time.Duration
	lastAuto     time.Time
}

// SaveData — сериализуемое состояние игры.
type SaveData struct {
	Version   string                 `json:"version"`
	Timestamp int64                  `json:"timestamp"`
	Slot      int                    `json:"slot"`
	SceneName string                 `json:"sceneName"`
	Player    map[string]interface{} `json:"player,omitempty"`
	World     map[string]interface{} `json:"world,omitempty"`
	Custom    map[string]interface{} `json:"custom,omitempty"`
}

// NewSaveSystem создаёт систему сохранений.
func NewSaveSystem(saveDir, slotPrefix string, maxSlots int) *SaveSystem {
	if saveDir == "" {
		saveDir = "saves"
	}
	if slotPrefix == "" {
		slotPrefix = "slot"
	}
	if maxSlots <= 0 {
		maxSlots = 10
	}
	return &SaveSystem{
		saveDir:      saveDir,
		slotPrefix:   slotPrefix,
		maxSlots:     maxSlots,
		autoSave:     false,
		autoInterval: 5 * time.Minute,
	}
}

// EnableAutoSave включает автосохранение с заданным интервалом.
func (s *SaveSystem) EnableAutoSave(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.autoSave = true
	s.autoInterval = interval
}

// DisableAutoSave отключает автосохранение.
func (s *SaveSystem) DisableAutoSave() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.autoSave = false
}

// Save записывает данные в слот.
func (s *SaveSystem) Save(data *SaveData, slot int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if slot < 0 || slot >= s.maxSlots {
		slot = 0
	}

	if err := os.MkdirAll(s.saveDir, 0755); err != nil {
		return err
	}

	data.Timestamp = time.Now().Unix()
	data.Slot = slot
	if data.Version == "" {
		data.Version = "1.0"
	}

	path := filepath.Join(s.saveDir, s.slotName(slot))
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, raw, 0644)
}

// Load загружает данные из слота.
func (s *SaveSystem) Load(slot int) (*SaveData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if slot < 0 || slot >= s.maxSlots {
		slot = 0
	}

	path := filepath.Join(s.saveDir, s.slotName(slot))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var data SaveData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// Exists проверяет, есть ли сохранение в слоте.
func (s *SaveSystem) Exists(slot int) bool {
	if slot < 0 || slot >= s.maxSlots {
		return false
	}
	path := filepath.Join(s.saveDir, s.slotName(slot))
	_, err := os.Stat(path)
	return err == nil
}

// Delete удаляет сохранение в слоте.
func (s *SaveSystem) Delete(slot int) error {
	if slot < 0 || slot >= s.maxSlots {
		return nil
	}
	path := filepath.Join(s.saveDir, s.slotName(slot))
	return os.Remove(path)
}

// List возвращает информацию о всех слотах.
func (s *SaveSystem) List() []SlotInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]SlotInfo, s.maxSlots)
	for i := 0; i < s.maxSlots; i++ {
		path := filepath.Join(s.saveDir, s.slotName(i))
		fi, err := os.Stat(path)
		out[i] = SlotInfo{Slot: i, Exists: err == nil}
		if err == nil {
			out[i].Modified = fi.ModTime()
		}
	}
	return out
}

// SlotInfo — информация о слоте сохранения.
type SlotInfo struct {
	Slot     int       `json:"slot"`
	Exists   bool      `json:"exists"`
	Modified time.Time `json:"modified,omitempty"`
}

// TryAutoSave вызывать каждый кадр/тик; сохраняет по таймеру.
func (s *SaveSystem) TryAutoSave(builder func() *SaveData, slot int) (saved bool, err error) {
	s.mu.Lock()
	if !s.autoSave || s.autoInterval <= 0 {
		s.mu.Unlock()
		return false, nil
	}
	now := time.Now()
	if now.Sub(s.lastAuto) < s.autoInterval {
		s.mu.Unlock()
		return false, nil
	}
	s.lastAuto = now
	s.mu.Unlock()

	data := builder()
	if data == nil {
		return false, nil
	}
	err = s.Save(data, slot)
	return err == nil, err
}

func (s *SaveSystem) slotName(slot int) string {
	return fmt.Sprintf("%s%d.json", s.slotPrefix, slot)
}
