package render

import (
	"testing"
)

func TestBuildHUDBarAndLines(t *testing.T) {
	o := &HUDOverlay{
		HealthFill: 0.75,
		Lines:      []string{"Health: 75  Orbs: 2/8", "R — restart"},
	}
	bars, text := BuildHUD(o, 1280, 720)

	if len(bars) == 0 || len(bars)%hudFloatsPerVert != 0 {
		t.Fatalf("barVerts: %d floats, кратно %d", len(bars), hudFloatsPerVert)
	}
	// 3 квада: подложка + заливка (полоса) — по 6 вершин
	if quads := len(bars) / hudFloatsPerVert / 6; quads != 2 {
		t.Fatalf("полоса: %d квадов, ожидалось 2", quads)
	}
	// Текст: по 6 вершин на глиф
	verts := len(text) / hudFloatsPerVert
	if verts%6 != 0 || verts == 0 {
		t.Fatalf("textVerts: %d вершин, не кратно 6", verts)
	}

	checkBounds := func(name string, v []float32) {
		for i := 0; i+1 < len(v); i += hudFloatsPerVert {
			if v[i] < 0 || v[i] > 1280 || v[i+1] < 0 || v[i+1] > 720 {
				t.Fatalf("%s: вершина (%f,%f) вне экрана", name, v[i], v[i+1])
			}
		}
	}
	checkBounds("bars", bars)
	checkBounds("text", text)
}

func TestBuildHUDCenterText(t *testing.T) {
	o := &HUDOverlay{CenterText: "VICTORY!", SubText: "ENTER — next level"}
	bars, text := BuildHUD(o, 1280, 720)

	// Подложка под центральный текст
	if len(bars) != 6*hudFloatsPerVert {
		t.Fatalf("центральная подложка: %d вершин, ожидалось 6", len(bars)/hudFloatsPerVert)
	}
	// Центрирование: подложка симметрична относительно 640
	x0, x1 := bars[0], bars[hudFloatsPerVert]
	if (x0+x1)/2 != 640 {
		t.Fatalf("подложка не по центру: x0=%f x1=%f", x0, x1)
	}
	if len(text) == 0 {
		t.Fatal("глифы не построены")
	}
}

func TestBuildHUDNilAndEmpty(t *testing.T) {
	if b, txt := BuildHUD(nil, 100, 100); b != nil || txt != nil {
		t.Fatal("nil overlay должен давать пустые вершины")
	}
	if b, txt := BuildHUD(&HUDOverlay{}, 100, 100); b != nil || txt != nil {
		t.Fatal("пустой overlay должен давать пустые вершины")
	}
	// HealthFill > 1 обрезается
	b, _ := BuildHUD(&HUDOverlay{HealthFill: 5}, 1000, 1000)
	w := b[hudFloatsPerVert*6] - b[0] // ширина заливки (второй квад)
	if w > 258 {
		t.Fatalf("заливка шире полосы: %f", w)
	}
}
