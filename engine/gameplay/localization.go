package gameplay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Localization — простой key-value локализатор (инди: смена языка, таблицы строк).
type Localization struct {
	mu       sync.RWMutex
	locale   string
	strings  map[string]string
	fallback string
}

// NewLocalization создаёт локализатор.
func NewLocalization(locale, fallback string) *Localization {
	if fallback == "" {
		fallback = "en"
	}
	return &Localization{
		locale:   locale,
		strings:  make(map[string]string),
		fallback: fallback,
	}
}

// LoadFile загружает JSON вида { "key": "value", ... }.
func (l *Localization) LoadFile(path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	for k, v := range m {
		l.strings[k] = v
	}
	return nil
}

// Clear очищает таблицу строк (перед загрузкой другой локали).
func (l *Localization) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.strings = make(map[string]string)
}

// LoadDir загружает все .json из папки (например locales/en.json, locales/ru.json).
func (l *Localization) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".json" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if err := l.LoadFile(path); err != nil {
			return err
		}
	}
	return nil
}

// SetLocale переключает локаль (файлы можно грузить отдельно под каждую).
func (l *Localization) SetLocale(locale string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.locale = locale
}

// Get возвращает строку по ключу, иначе key.
func (l *Localization) Get(key string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if s, ok := l.strings[key]; ok {
		return s
	}
	return key
}

// Has проверяет наличие ключа.
func (l *Localization) Has(key string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, ok := l.strings[key]
	return ok
}
