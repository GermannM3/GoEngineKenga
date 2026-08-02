package gameplay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Localization — i18n локализатор для инди-игр (key-value, fallback, printf).
type Localization struct {
	mu       sync.RWMutex
	locale   string
	fallback string
	// locales[locale][key] = value
	locales map[string]map[string]string
}

// NewLocalization создаёт локализатор.
func NewLocalization(locale, fallback string) *Localization {
	if fallback == "" {
		fallback = "en"
	}
	return &Localization{
		locale:   locale,
		fallback: fallback,
		locales:  make(map[string]map[string]string),
	}
}

// LoadFile загружает JSON вида { "key": "value", ... } в текущую локаль.
func (l *Localization) LoadFile(path string) error {
	return l.LoadLocaleFile(l.locale, path)
}

// LoadLocaleFile загружает строки для указанной локали (например locales/en.json).
func (l *Localization) LoadLocaleFile(locale, path string) error {
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
	if l.locales[locale] == nil {
		l.locales[locale] = make(map[string]string)
	}
	for k, v := range m {
		l.locales[locale][k] = v
	}
	return nil
}

// LoadDir загружает все locales/{locale}.json из папки (en.json, ru.json, ...).
func (l *Localization) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		locale := strings.TrimSuffix(name, ".json")
		path := filepath.Join(dir, name)
		if err := l.LoadLocaleFile(locale, path); err != nil {
			return err
		}
	}
	return nil
}

// Clear очищает все загруженные локали.
func (l *Localization) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.locales = make(map[string]map[string]string)
}

// SetLocale переключает активную локаль.
func (l *Localization) SetLocale(locale string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.locale = locale
}

// Locale возвращает текущую локаль.
func (l *Localization) Locale() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.locale
}

// Get возвращает строку по ключу (сначала текущая локаль, затем fallback), иначе key.
func (l *Localization) Get(key string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if m := l.locales[l.locale]; m != nil {
		if s, ok := m[key]; ok {
			return s
		}
	}
	if m := l.locales[l.fallback]; m != nil {
		if s, ok := m[key]; ok {
			return s
		}
	}
	return key
}

// Tr форматирует строку с подстановкой (аналог fmt.Sprintf): Tr("score", 100) → "Score: 100".
func (l *Localization) Tr(key string, args ...any) string {
	format := l.Get(key)
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

// Has проверяет наличие ключа в текущей или fallback локали.
func (l *Localization) Has(key string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if m := l.locales[l.locale]; m != nil {
		if _, ok := m[key]; ok {
			return true
		}
	}
	if m := l.locales[l.fallback]; m != nil {
		_, ok := m[key]
		return ok
	}
	return false
}

// AvailableLocales возвращает список загруженных локальей.
func (l *Localization) AvailableLocales() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]string, 0, len(l.locales))
	for loc := range l.locales {
		out = append(out, loc)
	}
	return out
}
