package ebiten

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	_ "image/jpeg"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/ecs"
	"goenginekenga/engine/input"
	"goenginekenga/engine/render"
	"goenginekenga/engine/ui"
)

//go:embed logo.jpg
var logoBytes []byte

type Backend struct {
	title  string
	width  int
	height int

	frame *render.Frame

	angle float64

	OnUpdate func(dtSeconds float64)

	resolver *asset.Resolver
	logf     func(format string, args ...any)

	// Splash screen
	splashImage   *ebiten.Image
	splashSeconds float64
	splashBgColor color.RGBA

	// Input
	InputState *input.State

	// UI
	UIContext *ui.UIRenderContext

	// 3D Renderer
	renderer3D  *Renderer3D
	use3DRender bool
}

func New(title string, width, height int) *Backend {
	return &Backend{
		title:       title,
		width:       width,
		height:      height,
		InputState:  input.NewState(),
		UIContext:   ui.NewUIRenderContext(),
		renderer3D:  NewRenderer3D(width, height),
		use3DRender: true, // Enable 3D rendering by default
	}
}

// Enable3D enables or disables 3D rendering
func (b *Backend) Enable3D(enabled bool) {
	b.use3DRender = enabled
}

// GetRenderer3D returns the 3D renderer
func (b *Backend) GetRenderer3D() *Renderer3D {
	return b.renderer3D
}

func (b *Backend) RunLoop(initial *render.Frame) error {
	b.frame = initial
	if initial != nil && initial.ProjectDir != "" {
		if r, err := asset.NewResolver(initial.ProjectDir); err == nil {
			b.resolver = r
		}
	}
	if b.logf == nil {
		b.logf = func(string, ...any) {}
	}

	// Инициализация splash screen
	b.splashSeconds = 2.5                                        // показываем лого 2.5 секунды
	b.splashBgColor = color.RGBA{R: 245, G: 240, B: 230, A: 255} // кремовый фон под лого

	// Декодируем логотип из embedded bytes
	if len(logoBytes) > 0 {
		if img, _, err := image.Decode(bytes.NewReader(logoBytes)); err == nil {
			b.splashImage = ebiten.NewImageFromImage(img)
		}
	}

	ebiten.SetWindowTitle(b.title)
	ebiten.SetWindowSize(b.width, b.height)
	return ebiten.RunGame(b)
}

func (b *Backend) Update() error {
	dt := 1.0 / 60.0

	// Во время сплэша не обновляем игру
	if b.splashSeconds > 0 {
		b.splashSeconds -= dt
		return nil
	}

	// Poll input
	b.pollInput()

	if b.OnUpdate != nil {
		b.OnUpdate(dt)
	}
	b.angle += dt

	// End frame for input (store previous state)
	b.InputState.EndFrame()

	return nil
}

// pollInput reads current input state from Ebiten
func (b *Backend) pollInput() {
	// Update mouse position
	mx, my := ebiten.CursorPosition()
	b.InputState.SetMousePosition(mx, my)

	// Update mouse buttons
	b.InputState.SetMouseButton(input.MouseButtonLeft, ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft))
	b.InputState.SetMouseButton(input.MouseButtonMiddle, ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle))
	b.InputState.SetMouseButton(input.MouseButtonRight, ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight))

	// Update mouse scroll
	scrollX, scrollY := ebiten.Wheel()
	b.InputState.SetMouseScroll(scrollX, scrollY)

	// Update keyboard
	for _, ek := range input.AllEbitenKeys {
		k := input.EbitenKeyToKey(ek)
		if k >= 0 {
			b.InputState.SetKeyPressed(k, ebiten.IsKeyPressed(ek))
		}
	}

	// Calculate deltas
	b.InputState.Update()
}

func (b *Backend) Draw(screen *ebiten.Image) {
	// Сплэш-экран с логотипом движка
	if b.splashSeconds > 0 && b.splashImage != nil {
		b.drawSplash(screen)
		return
	}

	cc := color.RGBA{R: 15, G: 18, B: 24, A: 255}
	if b.frame != nil {
		cc = b.frame.ClearColor
	}

	// Use 3D renderer if enabled
	if b.use3DRender && b.renderer3D != nil {
		var world *ecs.World
		if b.frame != nil {
			world = b.frame.World
		}
		b.renderer3D.DrawToScreen(screen, world, b.resolver, cc)
	} else {
		// Fallback to 2D wireframe rendering
		screen.Fill(cc)

		drawnMesh := false
		if b.frame != nil && b.frame.World != nil && b.resolver != nil {
			drawWireframe(screen, b.frame.World, b.resolver, b.logf)
			for _, id := range b.frame.World.Entities() {
				if mr, ok := b.frame.World.GetMeshRenderer(id); ok && mr.MeshAssetID != "" {
					drawnMesh = true
					break
				}
			}
		}
		if !drawnMesh {
			drawTestTriangle(screen, b.angle)
		}
	}

	// Debug overlay
	if b.frame != nil && b.frame.World != nil {
		mode := "3D"
		if !b.use3DRender {
			mode = "2D"
		}
		ebitenutil.DebugPrint(screen, "GoEngineKenga v0 ["+mode+"]\nEntities: "+itoa(len(b.frame.World.Entities())))
	}

	// Render UI on top
	if b.UIContext != nil {
		mousePressed := b.InputState.IsMouseButtonPressed(input.MouseButtonLeft)
		b.UIContext.Update(b.InputState.MouseX, b.InputState.MouseY, mousePressed)
		b.UIContext.Render(screen)
	}
}

// drawSplash рисует экран-заставку с логотипом движка
func (b *Backend) drawSplash(screen *ebiten.Image) {
	// Кремовый фон
	screen.Fill(b.splashBgColor)

	if b.splashImage == nil {
		return
	}

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	iw, ih := b.splashImage.Bounds().Dx(), b.splashImage.Bounds().Dy()

	// Масштабируем логотип: максимум 60% от меньшей стороны экрана
	maxSize := float64(min(sw, sh)) * 0.6
	scale := maxSize / float64(max(iw, ih))
	if scale > 1 {
		scale = 1 // не увеличиваем, если лого меньше
	}

	scaledW := float64(iw) * scale
	scaledH := float64(ih) * scale

	// Центрируем
	x := (float64(sw) - scaledW) / 2
	y := (float64(sh) - scaledH) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)

	screen.DrawImage(b.splashImage, op)

	// Текст "Powered by GoEngineKenga" внизу
	msg := "Powered by GoEngineKenga"
	ebitenutil.DebugPrintAt(screen, msg, sw/2-len(msg)*3, sh-30)
}

func drawTestTriangle(screen *ebiten.Image, angle float64) {
	w, h := screen.Size()
	cx, cy := float64(w)/2, float64(h)/2
	r := math.Min(float64(w), float64(h)) * 0.25

	p0x, p0y := cx+math.Cos(angle)*r, cy+math.Sin(angle)*r
	p1x, p1y := cx+math.Cos(angle+2.094)*r, cy+math.Sin(angle+2.094)*r
	p2x, p2y := cx+math.Cos(angle+4.188)*r, cy+math.Sin(angle+4.188)*r

	lineColor := color.RGBA{R: 180, G: 220, B: 255, A: 255}
	ebitenutil.DrawLine(screen, p0x, p0y, p1x, p1y, lineColor)
	ebitenutil.DrawLine(screen, p1x, p1y, p2x, p2y, lineColor)
	ebitenutil.DrawLine(screen, p2x, p2y, p0x, p0y, lineColor)
}

func itoa(v int) string {
	// локальный мини-itoa чтобы не тащить strconv в каждый draw
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [32]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func (b *Backend) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

// SetUIManager sets the UI manager for this backend
func (b *Backend) SetUIManager(uiManager *ui.UIManager) {
	if b.UIContext != nil {
		b.UIContext.SetUIManager(uiManager)
	}
}
