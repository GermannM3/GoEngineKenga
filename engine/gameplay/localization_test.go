package gameplay

import (
	"path/filepath"
	"testing"
)

func TestLocalizationLoadAndTranslate(t *testing.T) {
	loc := NewLocalization("ru", "en")
	dir := filepath.Join("..", "..", "samples", "cyber_ninja", "locales")
	if err := loc.LoadDir(dir); err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if loc.Locale() != "ru" {
		t.Fatalf("locale=%q, want ru", loc.Locale())
	}

	// Прямой перевод с подстановкой
	if got := loc.Tr("hud.health", 75); got != "Здоровье: 75" {
		t.Fatalf("hud.health ru: %q", got)
	}
	if got := loc.Tr("hud.items", 2, 3); got != "Сферы: 2/3" {
		t.Fatalf("hud.items ru: %q", got)
	}

	// Неизвестный ключ возвращается как есть
	if got := loc.Tr("no.such.key"); got != "no.such.key" {
		t.Fatalf("missing key: %q", got)
	}

	// Неизвестная локаль — fallback на en
	loc.SetLocale("de")
	if got := loc.Tr("hud.health", 50); got != "Health: 50" {
		t.Fatalf("fallback en: %q", got)
	}

	// Настройка пакетного локализатора (используется HUD через Tr())
	SetDefaultLocalization(loc)
	t.Cleanup(func() { SetDefaultLocalization(nil) })
	if Locale() != "de" {
		t.Fatalf("Locale()=%q, want de", Locale())
	}
	if got := Tr("hud.health", 25); got != "Health: 25" {
		t.Fatalf("package Tr fallback: %q", got)
	}
}

func TestLocalizationWithoutSetup(t *testing.T) {
	// Без пакетного локализатора Tr возвращает ключ, Locale — пустую строку
	if Locale() != "" {
		t.Fatalf("Locale()=%q, want empty", Locale())
	}
	if got := Tr("hud.health", 10); got != "hud.health" {
		t.Fatalf("Tr without setup: %q", got)
	}
}
