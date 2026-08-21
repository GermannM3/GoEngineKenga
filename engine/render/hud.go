package render

// Графический HUD: данные и построение вершин для бэкендов с 2D-оверлеем
// (WebGPU; Ebiten рисует свой HUD напрямую). Лейаут чистый, без GPU —
// тестируется обычным go test. Вершины: X,Y (пиксели), U,V (атлас), R,G,B,A.

// HUDOverlay — что показать поверх 3D-сцены.
type HUDOverlay struct {
	HealthFill float32  // заполнение полосы здоровья, 0..1 (0 — не показывать)
	Lines      []string // строки текста в левом верхнем углу
	CenterText string   // крупный текст по центру (победа/поражение)
	SubText    string   // подстрочник под центральным текстом
}

// Метрики атласа шрифта (basicfont.Face7x13): глиф 7×13 в ячейке 8×16.
const (
	HUDFirstChar = 32 // первый символ атласа (пробел)
	HUDCharCount = 95 // ASCII 32..126
	HUDCellW     = 8  // шаг символа по горизонтали в атласе
	HUDGlyphH    = 13 // высота глифа
	HUDAtlasW    = HUDCharCount * HUDCellW
	HUDAtlasH    = 16

	hudFloatsPerVert = 8 // X,Y,U,V,R,G,B,A
)

// BuildHUD строит вершины HUD-квадов в пиксельных координатах экрана:
// barVerts — полоса здоровья и подложки (текстура — белая 1×1),
// textVerts — глифы (текстура — атлас шрифта).
func BuildHUD(o *HUDOverlay, screenW, screenH int) (barVerts, textVerts []float32) {
	if o == nil || screenW <= 0 || screenH <= 0 {
		return nil, nil
	}
	const margin = 20.0

	if o.HealthFill > 0 {
		barVerts = hudRect(barVerts, margin, margin, 264, 22, 0.07, 0.08, 0.10, 0.8)
		fill := o.HealthFill
		if fill > 1 {
			fill = 1
		}
		r, g, b := float32(0.25), float32(0.85), float32(0.35)
		switch {
		case fill <= 0.25:
			r, g, b = 0.9, 0.25, 0.2
		case fill <= 0.5:
			r, g, b = 0.95, 0.8, 0.2
		}
		barVerts = hudRect(barVerts, margin+3, margin+3, 258*fill, 16, r, g, b, 1)
	}

	textVerts = hudLines(textVerts, o.Lines, margin, margin+34, 2)

	if o.CenterText != "" {
		scale := float32(5)
		tw := float32(len(o.CenterText)) * HUDCellW * scale
		th := HUDGlyphH * scale
		cx := (float32(screenW) - tw) / 2
		cy := float32(screenH)*0.36 - th/2
		barVerts = hudRect(barVerts, cx-28, cy-22, tw+56, th+44, 0.04, 0.05, 0.07, 0.7)
		textVerts = hudText(textVerts, o.CenterText, cx, cy, scale)
		if o.SubText != "" {
			textVerts = hudText(textVerts, o.SubText, (float32(screenW)-float32(len(o.SubText))*HUDCellW*2)/2, cy+th+26, 2)
		}
	}
	return barVerts, textVerts
}

// hudLines печатает строки с переносом по левому краю.
func hudLines(dst []float32, lines []string, x, y, scale float32) []float32 {
	lineH := HUDGlyphH*scale + 8
	for i, l := range lines {
		dst = hudText(dst, l, x, y+float32(i)*lineH, scale)
	}
	return dst
}

// hudText добавляет квады глифов строки (символы вне ASCII пропускаются).
func hudText(dst []float32, s string, x, y, scale float32) []float32 {
	r, g, b, a := float32(0.93), float32(0.95), float32(1.0), float32(1.0)
	for i, ch := range s {
		if ch < HUDFirstChar || ch >= HUDFirstChar+HUDCharCount {
			continue
		}
		dst = hudGlyph(dst, float32(x)+float32(i)*HUDCellW*scale, y, scale, int(ch-HUDFirstChar), r, g, b, a)
	}
	return dst
}

// hudGlyph — квад одного глифа из атласа (полу-пиксельный отступ от билинейного затекания).
func hudGlyph(dst []float32, x, y, scale float32, cell int, r, g, b, a float32) []float32 {
	u0 := (float32(cell*HUDCellW) + 0.5) / HUDAtlasW
	u1 := (float32(cell*HUDCellW+7) - 0.5) / HUDAtlasW
	v0 := float32(0.5) / HUDAtlasH
	v1 := float32(HUDGlyphH-0.5) / HUDAtlasH
	return hudQuad(dst, x, y, 7*scale, HUDGlyphH*scale, u0, v0, u1, v1, r, g, b, a)
}

// hudRect — непрозрачный цветной квад на всю UV.
func hudRect(dst []float32, x, y, w, h, r, g, b, a float32) []float32 {
	return hudQuad(dst, x, y, w, h, 0, 0, 1, 1, r, g, b, a)
}

// hudQuad — два треугольника (6 вершин): верхний левый угол (x,y).
func hudQuad(dst []float32, x, y, w, h, u0, v0, u1, v1, r, g, b, a float32) []float32 {
	x1, y1 := x+w, y+h
	type v = [8]float32
	row := func(px, py, pu, pv float32) v { return v{px, py, pu, pv, r, g, b, a} }
	p00 := row(x, y, u0, v0)
	p10 := row(x1, y, u1, v0)
	p11 := row(x1, y1, u1, v1)
	p01 := row(x, y1, u0, v1)
	dst = append(dst, p00[:]...)
	dst = append(dst, p11[:]...)
	dst = append(dst, p10[:]...)
	dst = append(dst, p00[:]...)
	dst = append(dst, p01[:]...)
	dst = append(dst, p11[:]...)
	return dst
}
