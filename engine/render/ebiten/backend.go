package ebiten

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/render"
)

type Backend struct {
	title  string
	width  int
	height int

	frame *render.Frame

	angle float64

	OnUpdate func(dtSeconds float64)

	resolver *asset.Resolver
	logf     func(format string, args ...any)
}

func New(title string, width, height int) *Backend {
	return &Backend{
		title:  title,
		width:  width,
		height: height,
	}
}

func (b *Backend) RunLoop(initial *render.Frame) error {
	b.frame = initial
	if initial != nil && initial.ProjectDir != "" {
		if r, err := asset.NewResolver(initial.ProjectDir); err == nil {
			b.resolver = r
		}
	}

	ebiten.SetWindowTitle(b.title)
	ebiten.SetWindowSize(b.width, b.height)
	return ebiten.RunGame(b)
}

func (b *Backend) Update() error {
	if b.OnUpdate != nil {
		b.OnUpdate(1.0 / 60.0)
	}
	b.angle += 1.0 / 60.0
	return nil
}

func (b *Backend) Draw(screen *ebiten.Image) {
	cc := color.RGBA{R: 15, G: 18, B: 24, A: 255}
	if b.frame != nil {
		cc = b.frame.ClearColor
	}
	screen.Fill(cc)

	// v0: если есть импортированные меши — рисуем wireframe; иначе — тестовый треугольник.
	b.logf("ebiten: drawing frame. World is nil? %v, Resolver is nil? %v\n", b.frame == nil || b.frame.World == nil, b.resolver == nil)
	drawnMesh := false
	if b.frame != nil && b.frame.World != nil && b.resolver != nil {
		b.logf("backend: calling drawWireframe\n")
		drawWireframe(screen, b.frame.World, b.resolver, b.logf)
		// эвристика: если в мире есть MeshRenderer с MeshAssetID — считаем, что пробовали рисовать меш
		for _, id := range b.frame.World.Entities() {
			if mr, ok := b.frame.World.GetMeshRenderer(id); ok && mr.MeshAssetID != "" {
				drawnMesh = true
				break
			}
		}
	}
	if !drawnMesh {
		b.logf("backend: drawing test triangle\n")
		drawTestTriangle(screen, b.angle)
	}

	// Debug overlay
	if b.frame != nil && b.frame.World != nil {
		ebitenutil.DebugPrint(screen, "GoEngineKenga v0\nEntities: "+itoa(len(b.frame.World.Entities())))
	}
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

