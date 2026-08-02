package ebiten

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"goenginekenga/engine/ecs"
)

// SpriteRenderSystem отвечает за отрисовку 2D-спрайтов
type SpriteRenderSystem struct {
	imageCache    map[string]*ebiten.Image
	projectDir    string
	fallbackImage *ebiten.Image
	tintCache     map[string]*ebiten.Image // tinted fallbacks per car color
}

// NewSpriteRenderSystem создает новую систему отрисовки спрайтов
func NewSpriteRenderSystem(projectDir string) *SpriteRenderSystem {
	return &SpriteRenderSystem{
		imageCache: make(map[string]*ebiten.Image),
		projectDir: projectDir,
		tintCache:  make(map[string]*ebiten.Image),
	}
}

type spriteDrawItem struct {
	entityID  ecs.EntityID
	sprite    ecs.SpriteRenderer
	transform ecs.Transform
}

// Render отрисовывает все спрайты в мире
func (s *SpriteRenderSystem) Render(screen *ebiten.Image, world *ecs.World) {
	if world == nil {
		return
	}
	entities := world.Entities()
	items := make([]spriteDrawItem, 0, len(entities))
	for _, id := range entities {
		sprite, hasSprite := world.GetSpriteRenderer(id)
		if !hasSprite || !sprite.Visible {
			continue
		}
		transform, hasTransform := world.GetTransform(id)
		if !hasTransform {
			continue
		}
		items = append(items, spriteDrawItem{entityID: id, sprite: sprite, transform: transform})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].sprite.Layer < items[j].sprite.Layer
	})

	camX, camY, camZoom := s.cameraView(world, entities)
	sw := float64(screen.Bounds().Dx())
	sh := float64(screen.Bounds().Dy())
	screenCenterX := sw / 2
	screenCenterY := sh / 2

	for _, item := range items {
		sprite := item.sprite
		tr := item.transform

		img, err := s.loadImage(sprite.TexturePath)
		if err != nil {
			img = s.coloredFallback(sprite)
		}

		imgW := float64(img.Bounds().Dx())
		imgH := float64(img.Bounds().Dy())
		scaleX := float64(tr.Scale.X)
		scaleY := float64(tr.Scale.Y)
		if scaleX == 0 {
			scaleX = 1
		}
		if scaleY == 0 {
			scaleY = 1
		}

		// Source rect / animation frame
		drawImg := img
		srcW, srcH := imgW, imgH
		if animImg, aw, ah, ok := s.animationSubImage(world, item.entityID, sprite, img); ok {
			drawImg = animImg
			srcW, srcH = aw, ah
		} else if sprite.SrcW > 0 && sprite.SrcH > 0 {
			r := image.Rect(sprite.SrcX, sprite.SrcY, sprite.SrcX+sprite.SrcW, sprite.SrcY+sprite.SrcH)
			if sub, ok := img.SubImage(r).(*ebiten.Image); ok {
				drawImg = sub
				srcW = float64(sprite.SrcW)
				srcH = float64(sprite.SrcH)
			}
		}

		opts := &ebiten.DrawImageOptions{}
		// Центр спрайта в начале координат, затем scale / flip / rotate, затем world→screen
		opts.GeoM.Translate(-srcW/2, -srcH/2)
		if sprite.FlipX {
			opts.GeoM.Scale(-1, 1)
		}
		if sprite.FlipY {
			opts.GeoM.Scale(1, -1)
		}
		opts.GeoM.Scale(scaleX, scaleY)
		// Rotation.Z — угол в градусах (0 = +X)
		if tr.Rotation.Z != 0 {
			opts.GeoM.Rotate(float64(tr.Rotation.Z) * math.Pi / 180)
		}

		relX := (float64(tr.Position.X) - camX) * camZoom
		relY := (float64(tr.Position.Y) - camY) * camZoom
		opts.GeoM.Translate(screenCenterX+relX, screenCenterY+relY)

		if sprite.ColorA > 0 {
			opts.ColorScale.Scale(
				float32(sprite.ColorR)/255,
				float32(sprite.ColorG)/255,
				float32(sprite.ColorB)/255,
				float32(sprite.ColorA)/255,
			)
		}

		screen.DrawImage(drawImg, opts)
	}
}

func (s *SpriteRenderSystem) cameraView(world *ecs.World, entities []ecs.EntityID) (camX, camY, zoom float64) {
	zoom = 1
	for _, id := range entities {
		cam, hasCam := world.GetCamera2D(id)
		if !hasCam {
			continue
		}
		zoom = float64(cam.Zoom)
		if zoom <= 0 {
			zoom = 1
		}
		// Follow by entity ID
		if cam.FollowID != 0 {
			if tr, ok := world.GetTransform(cam.FollowID); ok {
				return float64(tr.Position.X), float64(tr.Position.Y), zoom
			}
		}
		// Fallback: follow Player by name
		for _, eid := range entities {
			if world.Name(eid) == "Player" {
				if tr, ok := world.GetTransform(eid); ok {
					return float64(tr.Position.X), float64(tr.Position.Y), zoom
				}
			}
		}
		if tr, ok := world.GetTransform(id); ok {
			return float64(tr.Position.X), float64(tr.Position.Y), zoom
		}
		break
	}
	return 0, 0, 1
}

func (s *SpriteRenderSystem) animationSubImage(world *ecs.World, id ecs.EntityID, sprite ecs.SpriteRenderer, img *ebiten.Image) (*ebiten.Image, float64, float64, bool) {
	animState, hasAnimState := world.GetAnimationState(id)
	if !hasAnimState || !animState.IsPlaying {
		return nil, 0, 0, false
	}
	animCtrl, hasAnimCtrl := world.GetAnimationController(id)
	if !hasAnimCtrl {
		return nil, 0, 0, false
	}
	var clip *ecs.AnimationClip
	for i := range animCtrl.Clips {
		if animCtrl.Clips[i].Name == animState.CurrentClip {
			clip = &animCtrl.Clips[i]
			break
		}
	}
	if clip == nil || len(clip.Frames) == 0 || animState.CurrentFrame >= len(clip.Frames) {
		return nil, 0, 0, false
	}
	frameIdx := clip.Frames[animState.CurrentFrame]
	fw, fh := clip.FrameWidth, clip.FrameHeight
	if fw == 0 || fh == 0 {
		fw, fh = sprite.SrcW, sprite.SrcH
	}
	if fw == 0 || fh == 0 {
		return nil, 0, 0, false
	}
	srcX, srcY := clip.StartX, clip.StartY
	switch clip.Layout {
	case "vertical":
		srcX = clip.StartX + frameIdx*clip.StepX
		srcY = clip.StartY + frameIdx*clip.StepY
	case "grid":
		cols := img.Bounds().Dx() / fw
		if cols <= 0 {
			cols = 1
		}
		srcX = clip.StartX + (frameIdx%cols)*clip.StepX
		srcY = clip.StartY + (frameIdx/cols)*clip.StepY
	default:
		stepX := clip.StepX
		if stepX == 0 {
			stepX = fw
		}
		srcX = clip.StartX + frameIdx*stepX
		srcY = clip.StartY
	}
	r := image.Rect(srcX, srcY, srcX+fw, srcY+fh)
	sub, ok := img.SubImage(r).(*ebiten.Image)
	if !ok {
		return nil, 0, 0, false
	}
	return sub, float64(fw), float64(fh), true
}

func (s *SpriteRenderSystem) loadImage(texturePath string) (*ebiten.Image, error) {
	if img, exists := s.imageCache[texturePath]; exists {
		return img, nil
	}
	fullPath := filepath.Join(s.projectDir, filepath.FromSlash(texturePath))
	file, err := os.Open(fullPath)
	if err != nil {
		// fallback: без префикса assets/
		alt := filepath.Join(s.projectDir, "assets", filepath.FromSlash(texturePath))
		file, err = os.Open(alt)
		if err != nil {
			return nil, err
		}
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		_, _ = file.Seek(0, 0)
		img, _, err = image.Decode(file)
		if err != nil {
			return nil, err
		}
	}
	ebitenImg := ebiten.NewImageFromImage(img)
	s.imageCache[texturePath] = ebitenImg
	return ebitenImg, nil
}

func (s *SpriteRenderSystem) getFallbackImage() *ebiten.Image {
	if s.fallbackImage != nil {
		return s.fallbackImage
	}
	m := image.NewRGBA(image.Rect(0, 0, 48, 28))
	for y := 0; y < 28; y++ {
		for x := 0; x < 48; x++ {
			c := color.RGBA{R: 100, G: 110, B: 130, A: 255}
			if y < 4 || y > 23 || x < 4 || x > 43 {
				c = color.RGBA{R: 40, G: 40, B: 50, A: 255}
			}
			m.Set(x, y, c)
		}
	}
	s.fallbackImage = ebiten.NewImageFromImage(m)
	return s.fallbackImage
}

// coloredFallback — заглушка с tint из SpriteRenderer (машины разных цветов)
func (s *SpriteRenderSystem) coloredFallback(sprite ecs.SpriteRenderer) *ebiten.Image {
	key := "fb"
	if sprite.ColorA > 0 {
		key = string([]byte{sprite.ColorR, sprite.ColorG, sprite.ColorB})
	} else if sprite.TexturePath != "" {
		key = sprite.TexturePath
	}
	if img, ok := s.tintCache[key]; ok {
		return img
	}
	r, g, b := uint8(80), uint8(180), uint8(220)
	if sprite.ColorA > 0 {
		r, g, b = sprite.ColorR, sprite.ColorG, sprite.ColorB
	} else {
		switch {
		case contains(sprite.TexturePath, "atom"):
			r, g, b = 0, 200, 255
		case contains(sprite.TexturePath, "m70"):
			r, g, b = 255, 120, 40
		case contains(sprite.TexturePath, "m90"):
			r, g, b = 60, 80, 255
		case contains(sprite.TexturePath, "power"):
			r, g, b = 255, 220, 40
		case contains(sprite.TexturePath, "track"):
			r, g, b = 70, 75, 85
		}
	}
	m := image.NewRGBA(image.Rect(0, 0, 64, 36))
	for y := 0; y < 36; y++ {
		for x := 0; x < 64; x++ {
			c := color.RGBA{R: r, G: g, B: b, A: 255}
			// колёса
			if (x < 10 || x > 53) && y > 22 {
				c = color.RGBA{R: 20, G: 20, B: 25, A: 255}
			}
			// кабина
			if x > 18 && x < 46 && y > 6 && y < 20 {
				c = color.RGBA{R: uint8(float32(r) * 0.6), G: uint8(float32(g) * 0.6), B: uint8(float32(b) * 0.7), A: 255}
			}
			m.Set(x, y, c)
		}
	}
	img := ebiten.NewImageFromImage(m)
	s.tintCache[key] = img
	return img
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}

// ClearCache очищает кеш изображений
func (s *SpriteRenderSystem) ClearCache() {
	s.imageCache = make(map[string]*ebiten.Image)
	s.tintCache = make(map[string]*ebiten.Image)
	s.fallbackImage = nil
}
