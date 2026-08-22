//go:build webgpu && !js

package webgpu

import (
	"fmt"
	"math"

	"github.com/cogentcore/webgpu/wgpu"

	emath "goenginekenga/engine/math"
)

// Процедурное окружение для IBL: sky-кубомапа + diffused irradiance-кубомапа.
// Никаких ассетов не требует: градиент неба генерируется на CPU, свёртка
// освещения по полусфере делается заранее (8×8 на грань).
// Конвенция граней совпадает с point-shadow cubemap (см. buildPointFaceViewProj).

// cubeFaceDir возвращает направление для грани face и координат (u,v) ∈ [-1,1].
func cubeFaceDir(face int, u, v float32) emath.Vec3 {
	var d emath.Vec3
	switch face {
	case 0: // +X
		d = emath.Vec3{X: 1, Y: -v, Z: -u}
	case 1: // -X
		d = emath.Vec3{X: -1, Y: -v, Z: u}
	case 2: // +Y
		d = emath.Vec3{X: u, Y: 1, Z: v}
	case 3: // -Y
		d = emath.Vec3{X: u, Y: -1, Z: -v}
	case 4: // +Z
		d = emath.Vec3{X: u, Y: -v, Z: 1}
	case 5: // -Z
		d = emath.Vec3{X: -u, Y: -v, Z: -1}
	}
	return renderNormalize(d)
}

func renderNormalize(v emath.Vec3) emath.Vec3 {
	l := float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
	if l < 1e-6 {
		return emath.Vec3{X: 0, Y: 1, Z: 0}
	}
	return emath.Vec3{X: v.X / l, Y: v.Y / l, Z: v.Z / l}
}

// skyColor возвращает процедурный цвет неба по направлению.
func skyColor(dir emath.Vec3) (r, g, b float32) {
	// Солнце для блика на окружении.
	sun := emath.Vec3{X: 0.45, Y: 0.82, Z: 0.32}
	sunGlow := float32(math.Pow(float64(max32(0, dir.X*sun.X+dir.Y*sun.Y+dir.Z*sun.Z)), 48)) * 5.0

	t := dir.Y
	var cr, cg, cb float32
	if t >= 0 {
		// Зенит → горизонт: синеватая дымка.
		zen := t * t * 0.55
		cr = 0.42 + zen*0.20
		cg = 0.58 + zen*0.12
		cb = 0.86 + zen*0.08
	} else {
		// Ниже горизонта: тёмная земля.
		gr := max32(0, -t)
		cr = 0.10 - gr*0.05
		cg = 0.09 - gr*0.045
		cb = 0.08 - gr*0.04
	}
	return cr + sunGlow*0.8, cg + sunGlow*0.75, cb + sunGlow*0.6
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// generateEnvData строит RGBA8-данные кубомапы size×size×6.
func generateEnvData(size int) []byte {
	data := make([]byte, size*size*6*4)
	for f := 0; f < 6; f++ {
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				u := (float32(x)+0.5)/float32(size)*2 - 1
				v := (float32(y)+0.5)/float32(size)*2 - 1
				dir := cubeFaceDir(f, u, v)
				r, g, b := skyColor(dir)
				idx := (f*size*size + y*size + x) * 4
				data[idx] = byte(clamp01(r) * 255)
				data[idx+1] = byte(clamp01(g) * 255)
				data[idx+2] = byte(clamp01(b) * 255)
				data[idx+3] = 255
			}
		}
	}
	return data
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// envLookup возвращает цвет окружения по направлению (та же конвенция граней).
func envLookup(env []byte, size int, dir emath.Vec3) (r, g, b float32) {
	ax, ay, az := abs32(dir.X), abs32(dir.Y), abs32(dir.Z)
	face, u, v := 0, float32(0), float32(0)
	switch {
	case ax >= ay && ax >= az:
		if dir.X >= 0 {
			face, u, v = 0, -dir.Z, -dir.Y
		} else {
			face, u, v = 1, dir.Z, -dir.Y
		}
	case ay >= ax && ay >= az:
		if dir.Y >= 0 {
			face, u, v = 2, dir.X, dir.Z
		} else {
			face, u, v = 3, dir.X, -dir.Z
		}
	default:
		if dir.Z >= 0 {
			face, u, v = 4, dir.X, -dir.Y
		} else {
			face, u, v = 5, -dir.X, -dir.Y
		}
	}
	sx := int(clamp01((u+1)*0.5) * float32(size-1))
	sy := int(clamp01((v+1)*0.5) * float32(size-1))
	idx := (face*size*size + sy*size + sx) * 4
	return float32(env[idx]) / 255, float32(env[idx+1]) / 255, float32(env[idx+2]) / 255
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// generateIrradianceData свёртка освещения по полусфере: для каждого текселя
// irradiance-кубомапы усредняем окружение по cos-взвешенным выборкам.
func generateIrradianceData(env []byte, envSize, irrSize int) []byte {
	const samples = 128
	dirs := make([]emath.Vec3, samples)
	for i := 0; i < samples; i++ {
		// Сферическая фибоначчи-решётка: равномерно по полусфере (z >= 0 локально).
		golden := float32(2.399963) // pi*(3-sqrt5)
		yi := 1 - float32(i)/(samples-1)
		theta := float32(math.Acos(float64(yi)))
		phi := golden * float32(i)
		dirs[i] = emath.Vec3{
			X: float32(math.Sin(float64(theta))) * float32(math.Cos(float64(phi))),
			Y: yi,
			Z: float32(math.Sin(float64(theta))) * float32(math.Sin(float64(phi))),
		}
	}

	data := make([]byte, irrSize*irrSize*6*4)
	for f := 0; f < 6; f++ {
		for y := 0; y < irrSize; y++ {
			for x := 0; x < irrSize; x++ {
				u := (float32(x)+0.5)/float32(irrSize)*2 - 1
				v := (float32(y)+0.5)/float32(irrSize)*2 - 1
				n := cubeFaceDir(f, u, v)

				var tr, tg, tb, wsum float32
				for _, s := range dirs {
					w := max32(0, s.X*n.X+s.Y*n.Y+s.Z*n.Z)
					if w <= 0 {
						continue
					}
					r, g, b := envLookup(env, envSize, s)
					tr += r * w
					tg += g * w
					tb += b * w
					wsum += w
				}
				if wsum > 0 {
					tr, tg, tb = tr/wsum, tg/wsum, tb/wsum
				}
				idx := (f*irrSize*irrSize + y*irrSize + x) * 4
				data[idx] = byte(clamp01(tr) * 255)
				data[idx+1] = byte(clamp01(tg) * 255)
				data[idx+2] = byte(clamp01(tb) * 255)
				data[idx+3] = 255
			}
		}
	}
	return data
}

// createEnvTextures создаёт и заливает env/irradiance кубомапы.
func createEnvTextures(device *wgpu.Device, queue *wgpu.Queue) (env *wgpu.Texture, envView *wgpu.TextureView, irr *wgpu.Texture, irrView *wgpu.TextureView, err error) {
	upload := func(size int, data []byte) (*wgpu.Texture, error) {
		tex, err := device.CreateTexture(&wgpu.TextureDescriptor{
			Label:         "ibl cubemap",
			Size:          wgpu.Extent3D{Width: uint32(size), Height: uint32(size), DepthOrArrayLayers: 6},
			MipLevelCount: 1,
			SampleCount:   1,
			Dimension:     wgpu.TextureDimension2D,
			Format:        wgpu.TextureFormatRGBA8UnormSrgb,
			Usage:         wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst,
		})
		if err != nil {
			return nil, err
		}
		err = queue.WriteTexture(
			&wgpu.ImageCopyTexture{Texture: tex},
			data,
			&wgpu.TextureDataLayout{BytesPerRow: uint32(size) * 4, RowsPerImage: uint32(size)},
			&wgpu.Extent3D{Width: uint32(size), Height: uint32(size), DepthOrArrayLayers: 6},
		)
		if err != nil {
			tex.Release()
			return nil, err
		}
		return tex, nil
	}

	const envSize = 64
	const irrSize = 8
	env, err = upload(envSize, generateEnvData(envSize))
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("createEnvTextures env: %w", err)
	}
	irr, err = upload(irrSize, generateIrradianceData(generateEnvData(envSize), envSize, irrSize))
	if err != nil {
		env.Release()
		return nil, nil, nil, nil, fmt.Errorf("createEnvTextures irradiance: %w", err)
	}
	envView, err = env.CreateView(&wgpu.TextureViewDescriptor{
		Label:           "env cube view",
		Dimension:       wgpu.TextureViewDimensionCube,
		ArrayLayerCount: 6,
		MipLevelCount:   1,
	})
	if err != nil {
		irr.Release()
		env.Release()
		return nil, nil, nil, nil, err
	}
	irrView, err = irr.CreateView(&wgpu.TextureViewDescriptor{
		Label:           "irradiance cube view",
		Dimension:       wgpu.TextureViewDimensionCube,
		ArrayLayerCount: 6,
		MipLevelCount:   1,
	})
	if err != nil {
		envView.Release()
		irr.Release()
		env.Release()
		return nil, nil, nil, nil, err
	}
	return env, envView, irr, irrView, nil
}
