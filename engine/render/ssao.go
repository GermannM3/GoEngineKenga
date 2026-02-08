package render

import (
	"image"
	"math"
)

// SSAOParams настройки SSAO (Screen Space Ambient Occlusion)
type SSAOParams struct {
	Enabled   bool    // Включить SSAO
	Radius    int     // Радиус сэмплинга в пикселях (2..8)
	Samples   int     // Число сэмплов (8, 12, 16)
	Bias      float32 // Смещение для устранения артефактов (0.001..0.01)
	Strength  float32 // Сила затемнения (0.5..2.0)
	Downscale int     // Делитель разрешения (1=full, 2=half — быстрее)
}

// DefaultSSAOParams возвращает разумные настройки по умолчанию
func DefaultSSAOParams() SSAOParams {
	return SSAOParams{
		Enabled:   true,
		Radius:    4,
		Samples:   12,
		Bias:      0.005,
		Strength:  1.0,
		Downscale: 2, // половина разрешения — быстрее
	}
}

// Kernel — предвычисленные смещения для сэмплинга (hemisphere)
var ssaoKernel [16][2]float32

func init() {
	// Фиксированные смещения в круге (нормализованные)
	for i := 0; i < 16; i++ {
		angle := float64(i) * 0.3927 // ~22.5°
		ssaoKernel[i][0] = float32(math.Cos(angle))
		ssaoKernel[i][1] = float32(math.Sin(angle))
	}
}

// ApplySSAO применяет SSAO к буферу. Требует color и depth.
func ApplySSAO(color *image.RGBA, depth []float32, p SSAOParams) {
	if color == nil || depth == nil || !p.Enabled || p.Strength <= 0 {
		return
	}
	b := color.Bounds()
	w, h := b.Dx(), b.Dy()
	if w*h != len(depth) {
		return
	}
	ds := p.Downscale
	if ds < 1 {
		ds = 1
	}
	if ds > 4 {
		ds = 4
	}
	radius := p.Radius
	if radius < 1 {
		radius = 1
	}
	if radius > 12 {
		radius = 12
	}
	samples := p.Samples
	if samples > 16 {
		samples = 16
	}
	if samples < 4 {
		samples = 4
	}

	bias := p.Bias
	if bias <= 0 {
		bias = 0.001
	}

	// Вычисляем occlusion в уменьшенном разрешении
	sw := w / ds
	sh := h / ds
	if sw < 4 || sh < 4 {
		return
	}
	occlusion := make([]float32, sw*sh)

	for oy := 0; oy < sh; oy++ {
		for ox := 0; ox < sw; ox++ {
			cx := ox * ds
			cy := oy * ds
			if cx >= w || cy >= h {
				continue
			}
			centerIdx := cy*w + cx
			centerDepth := depth[centerIdx]
			if centerDepth >= 0.9999 {
				occlusion[oy*sw+ox] = 1.0
				continue
			}
			occ := float32(0)
			for s := 0; s < samples; s++ {
				dx := ssaoKernel[s][0] * float32(radius)
				dy := ssaoKernel[s][1] * float32(radius)
				sx := int(float32(cx) + dx)
				sy := int(float32(cy) + dy)
				if sx < 0 {
					sx = 0
				}
				if sx >= w {
					sx = w - 1
				}
				if sy < 0 {
					sy = 0
				}
				if sy >= h {
					sy = h - 1
				}
				sampleDepth := depth[sy*w+sx]
				if sampleDepth < centerDepth-bias && sampleDepth < 0.9999 {
					occ += 1.0
				}
			}
			occ /= float32(samples)
			occlusion[oy*sw+ox] = 1.0 - occ
		}
	}

	// Простой box blur для сглаживания
	blurRad := 1
	if radius >= 4 {
		blurRad = 2
	}
	smoothed := make([]float32, sw*sh)
	for oy := 0; oy < sh; oy++ {
		for ox := 0; ox < sw; ox++ {
			var sum float32
			count := 0
			for dy := -blurRad; dy <= blurRad; dy++ {
				for dx := -blurRad; dx <= blurRad; dx++ {
					sx := ox + dx
					sy := oy + dy
					if sx >= 0 && sx < sw && sy >= 0 && sy < sh {
						sum += occlusion[sy*sw+sx]
						count++
					}
				}
			}
			if count > 0 {
				smoothed[oy*sw+ox] = sum / float32(count)
			} else {
				smoothed[oy*sw+ox] = occlusion[oy*sw+ox]
			}
		}
	}

	// Применяем к цвету (затемняем по occlusion)
	str := p.Strength
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ox := x / ds
			oy := y / ds
			if ox >= sw {
				ox = sw - 1
			}
			if oy >= sh {
				oy = sh - 1
			}
			occ := smoothed[oy*sw+ox]
			factor := 1.0 - (1.0-occ)*str
			if factor < 0.1 {
				factor = 0.1
			}
			idx := (y*w + x) * 4
			r := float32(color.Pix[idx]) * factor
			g := float32(color.Pix[idx+1]) * factor
			b := float32(color.Pix[idx+2]) * factor
			if r > 255 {
				r = 255
			}
			if g > 255 {
				g = 255
			}
			if b > 255 {
				b = 255
			}
			color.Pix[idx] = uint8(r)
			color.Pix[idx+1] = uint8(g)
			color.Pix[idx+2] = uint8(b)
		}
	}
}
