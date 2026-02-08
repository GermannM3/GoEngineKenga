package render

import "image"

// BloomParams настройки bloom-постобработки
type BloomParams struct {
	Enabled   bool    // Включить bloom
	Threshold float32 // Порог яркости (0..1), выше — попадает в bloom
	Strength  float32 // Сила свечения (0..2)
	Radius    int     // Радиус размытия (1..6)
	Downscale int     // Делитель разрешения для bloom (1=full, 2=half, 4=quarter)
}

// DefaultBloomParams возвращает разумные настройки по умолчанию
func DefaultBloomParams() BloomParams {
	return BloomParams{
		Enabled:   true,
		Threshold: 0.65,
		Strength:  0.5,
		Radius:    3,
		Downscale: 2, // половина разрешения — быстрее
	}
}

// ApplyBloom применяет bloom к буферу. Модифицирует buf in-place.
func ApplyBloom(buf *image.RGBA, p BloomParams) {
	if buf == nil || !p.Enabled || p.Strength <= 0 || p.Radius < 1 {
		return
	}

	b := buf.Bounds()
	w, h := b.Dx(), b.Dy()
	ds := p.Downscale
	if ds < 1 {
		ds = 1
	}
	if ds > 4 {
		ds = 4
	}

	bw := w / ds
	bh := h / ds
	if bw < 4 || bh < 4 {
		return
	}
	bn := bw * bh

	thr := p.Threshold
	if thr < 0 {
		thr = 0
	}
	if thr > 1 {
		thr = 1
	}

	// 1. Extract bright pixels (downscaled)
	bloom := make([]float32, bn*3)
	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			var sumR, sumG, sumB float32
			count := 0
			for dy := 0; dy < ds && by*ds+dy < h; dy++ {
				for dx := 0; dx < ds && bx*ds+dx < w; dx++ {
					px := bx*ds + dx
					py := by*ds + dy
					idx := (py*w + px) * 4
					r := float32(buf.Pix[idx]) / 255
					g := float32(buf.Pix[idx+1]) / 255
					b := float32(buf.Pix[idx+2]) / 255
					lum := 0.2126*r + 0.7152*g + 0.0722*b
					if lum > thr {
						f := (lum - thr) / (1 - thr + 0.001)
						sumR += r * f
						sumG += g * f
						sumB += b * f
						count++
					}
				}
			}
			if count > 0 {
				inv := 1.0 / float32(count)
				i := (by*bw + bx) * 3
				bloom[i] = sumR * inv
				bloom[i+1] = sumG * inv
				bloom[i+2] = sumB * inv
			}
		}
	}

	// 2. Box blur (на уменьшенном буфере)
	radius := p.Radius
	if radius > 8 {
		radius = 8
	}
	blurred := make([]float32, bn*3)
	for pass := 0; pass < 2; pass++ {
		src := bloom
		dst := blurred
		if pass == 1 {
			src = blurred
			dst = bloom
		}
		boxBlur(src, dst, bw, bh, radius)
	}

	// 3. Upsample и add bloom к исходному
	str := p.Strength
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			bx := x / ds
			by := y / ds
			if bx >= bw {
				bx = bw - 1
			}
			if by >= bh {
				by = bh - 1
			}
			bi := (by*bw + bx) * 3
			idx := (y*w + x) * 4
			addR := bloom[bi] * str * 255
			addG := bloom[bi+1] * str * 255
			addB := bloom[bi+2] * str * 255
			r := int(buf.Pix[idx]) + int(addR)
			g := int(buf.Pix[idx+1]) + int(addG)
			b := int(buf.Pix[idx+2]) + int(addB)
			if r > 255 {
				r = 255
			}
			if g > 255 {
				g = 255
			}
			if b > 255 {
				b = 255
			}
			buf.Pix[idx] = uint8(r)
			buf.Pix[idx+1] = uint8(g)
			buf.Pix[idx+2] = uint8(b)
		}
	}
}

func boxBlur(src, dst []float32, w, h, radius int) {
	if radius < 1 {
		copy(dst, src)
		return
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sumR, sumG, sumB float32
			count := 0
			for dy := -radius; dy <= radius; dy++ {
				for dx := -radius; dx <= radius; dx++ {
					px := x + dx
					py := y + dy
					if px >= 0 && px < w && py >= 0 && py < h {
						i := (py*w + px) * 3
						sumR += src[i]
						sumG += src[i+1]
						sumB += src[i+2]
						count++
					}
				}
			}
			if count > 0 {
				inv := 1.0 / float32(count)
				i := (y*w + x) * 3
				dst[i] = sumR * inv
				dst[i+1] = sumG * inv
				dst[i+2] = sumB * inv
			}
		}
	}
}

