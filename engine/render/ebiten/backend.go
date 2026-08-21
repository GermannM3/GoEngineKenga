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
	"goenginekenga/engine/gameplay"
	"goenginekenga/engine/input"
	emath "goenginekenga/engine/math"
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

	// 2D Renderer
	spriteRenderer *SpriteRenderSystem
	use2DSprites   bool

	// Orbit camera (ПКМ rotate, СКМ pan, scroll zoom)
	orbitState   render.OrbitState
	orbitEnabled bool
	orbitSynced  bool
}

func New(title string, width, height int) *Backend {
	return &Backend{
		title:          title,
		width:          width,
		height:         height,
		InputState:     input.NewState(),
		UIContext:      ui.NewUIRenderContext(),
		renderer3D:     NewRenderer3D(width, height),
		use3DRender:    true,
		spriteRenderer: nil, // Будет инициализирована при первом использовании
		use2DSprites:   false,
		orbitState:     render.DefaultOrbitState(),
		orbitEnabled:   true,
	}
}

// Enable3D enables or disables 3D rendering
func (b *Backend) Enable3D(enabled bool) {
	b.use3DRender = enabled
}

// Enable2DSprites enables or disables 2D sprite rendering
func (b *Backend) Enable2DSprites(enabled bool) {
	b.use2DSprites = enabled
}

// HasSpriteRenderer проверяет, есть ли в сцене хотя бы один SpriteRenderer
func (b *Backend) HasSpriteRenderer(world *ecs.World) bool {
	if world == nil {
		return false
	}

	for _, id := range world.Entities() {
		if _, hasSprite := world.GetSpriteRenderer(id); hasSprite {
			return true
		}
	}
	return false
}

// GetSpriteRenderer возвращает систему отрисовки спрайтов
func (b *Backend) GetSpriteRenderer() *SpriteRenderSystem {
	return b.spriteRenderer
}

// SetSpriteRenderer устанавливает систему отрисовки спрайтов
func (b *Backend) SetSpriteRenderer(renderer *SpriteRenderSystem) {
	b.spriteRenderer = renderer
}

// EnableOrbitCamera включает/выключает orbit camera (ПКМ, СКМ, scroll)
func (b *Backend) EnableOrbitCamera(enabled bool) {
	b.orbitEnabled = enabled
}

// GetRenderer3D returns the 3D renderer
func (b *Backend) GetRenderer3D() *Renderer3D {
	return b.renderer3D
}

// SetFrame инициализирует backend для mobile (без RunGame).
func (b *Backend) SetFrame(initial *render.Frame) {
	b.frame = initial
	if initial != nil && initial.ProjectDir != "" {
		if r, ok := initial.Resolver.(*asset.Resolver); ok && r != nil {
			b.resolver = r
		} else if r, err := asset.NewResolver(initial.ProjectDir); err == nil {
			b.resolver = r
		}
	}
	if b.logf == nil {
		b.logf = func(string, ...any) {}
	}
	b.splashSeconds = 2.5
	b.splashBgColor = color.RGBA{R: 245, G: 240, B: 230, A: 255}
	if len(logoBytes) > 0 {
		if img, _, err := image.Decode(bytes.NewReader(logoBytes)); err == nil {
			b.splashImage = ebiten.NewImageFromImage(img)
		}
	}

	// Initialize sprite renderer if we have a project directory and need 2D sprites
	if initial != nil && initial.ProjectDir != "" && b.use2DSprites && b.spriteRenderer == nil {
		b.spriteRenderer = NewSpriteRenderSystem(initial.ProjectDir)
	}
}

func (b *Backend) RunLoop(initial *render.Frame) error {
	b.SetFrame(initial)
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

	// Пробрасываем ввод в Frame для систем (например, управление игроком)
	if b.frame != nil {
		b.frame.InputState = b.InputState
	}

	// Orbit camera: ПКМ orbit, СКМ pan, scroll zoom
	if b.orbitEnabled && b.use3DRender && b.frame != nil && b.frame.World != nil {
		b.updateOrbitCamera()
	}

	if b.frame != nil && b.frame.OnUpdate != nil {
		b.frame.OnUpdate(dt)
	}
	b.angle += dt

	// End frame for input (store previous state)
	b.InputState.EndFrame()

	return nil
}

// updateOrbitCamera применяет orbit/pan/zoom к первой камере в сцене.
// Следящие камеры (FollowID != 0) не трогаются — ими управляет runtime.UpdateFollowCameras.
func (b *Backend) updateOrbitCamera() {
	w := b.frame.World
	var camID ecs.EntityID
	var hasCam bool
	for _, id := range w.Entities() {
		if c, ok := w.GetCamera(id); ok {
			if c.FollowID != 0 {
				return // сцена со следящей камерой — orbit отключён
			}
			camID = id
			hasCam = true
			break
		}
	}
	if !hasCam {
		b.orbitSynced = false
		return
	}

	tr, hasTr := w.GetTransform(camID)
	if !hasTr {
		tr = ecs.Transform{Position: emath.Vec3{X: 0, Y: 5, Z: 10}, Scale: emath.Vec3{X: 1, Y: 1, Z: 1}}
	}

	// Синхронизация при первой камере, после смены сцены или по запросу
	if b.frame.OrbitResetRequested {
		b.frame.OrbitResetRequested = false
		b.orbitSynced = false
	}
	if !b.orbitSynced {
		b.orbitState.SyncFromTransform(tr.Position, tr.Rotation.Y, tr.Rotation.X)
		b.orbitSynced = true
	}

	// ПКМ: orbit
	if b.InputState.IsMouseButtonPressed(input.MouseButtonRight) {
		b.orbitState.Orbit(float32(b.InputState.MouseDeltaX), float32(b.InputState.MouseDeltaY))
	}
	// СКМ: pan
	if b.InputState.IsMouseButtonPressed(input.MouseButtonMiddle) {
		b.orbitState.Pan(float32(b.InputState.MouseDeltaX), float32(b.InputState.MouseDeltaY))
	}
	// Scroll: zoom
	if b.InputState.MouseScrollY != 0 {
		b.orbitState.Zoom(float32(b.InputState.MouseScrollY))
	}

	pos := b.orbitState.Position()
	tr.Position = pos
	tr.Rotation = emath.Vec3{X: b.orbitState.Pitch, Y: b.orbitState.Yaw, Z: tr.Rotation.Z}
	w.SetTransform(camID, tr)

	// Напрямую ставим Camera3D — минуя roundtrip через entity/updateCameraFromWorld
	cam := b.renderer3D.GetCamera()
	cam.SetPosition(pos)
	cam.SetTarget(b.orbitState.Target)
	b.renderer3D.CameraSetExternally = true
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

	// Check if we have sprites in the scene and auto-switch to 2D mode if needed
	if b.frame != nil && b.frame.World != nil {
		hasSprites := b.HasSpriteRenderer(b.frame.World)
		if hasSprites && !b.use3DRender {
			b.use2DSprites = true
		}
	}

	// Use 3D renderer if enabled
	if b.use3DRender && b.renderer3D != nil && !b.use2DSprites {
		var world *ecs.World
		if b.frame != nil {
			world = b.frame.World
		}
		b.renderer3D.DrawToScreen(screen, world, b.resolver, cc)
	} else if b.use2DSprites {
		screen.Fill(cc)
		var world *ecs.World
		if b.frame != nil {
			world = b.frame.World
		}
		if world != nil {
			if b.spriteRenderer == nil && b.frame != nil && b.frame.ProjectDir != "" {
				b.spriteRenderer = NewSpriteRenderSystem(b.frame.ProjectDir)
			}
			// Процедурный овал-трек под спрайтами
			if gameplay.HasKart(world) {
				b.drawKartTrack(screen, world)
			}
			if b.spriteRenderer != nil {
				b.spriteRenderer.Render(screen, world)
			}
		}
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

	// Debug / game HUD
	if b.frame != nil && b.frame.World != nil {
		w := b.frame.World
		mode := "3D"
		if !b.use3DRender {
			mode = "2D"
		}
		fps := ebiten.ActualFPS()
		msg := "GoEngineKenga [" + mode + "] Entities: " + itoa(len(w.Entities()))
		if fps > 0 {
			msg += " | FPS: " + itoa(int(fps))
		}
		if loc := gameplay.Locale(); loc != "" {
			msg += " | Loc: " + loc
		}
		if gameplay.HasKart(w) {
			msg += "\n" + b.kartHUD(w)
		} else if h := gameplay.GameHUD(); h != "" {
			msg += "\n" + h
		}
		msg += "\n"
		ebitenutil.DebugPrint(screen, msg)
		// Графический HUD (полоса здоровья, экраны победы/поражения) для 3D-игр.
		if !gameplay.HasKart(w) {
			b.drawGameHUD(screen)
		}
	}

	// Render UI on top
	if b.UIContext != nil {
		mousePressed := b.InputState.IsMouseButtonPressed(input.MouseButtonLeft)
		b.UIContext.Update(b.InputState.MouseX, b.InputState.MouseY, mousePressed)
		b.UIContext.Render(screen)
	}

	// Viewport stream для IDE (WebSocket)
	if b.frame != nil && b.frame.OnFrameRendered != nil {
		b.frame.OnFrameRendered(screen)
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

// drawKartTrack рисует овал трассы в экранных координатах (камера следует за игроком)
func (b *Backend) drawKartTrack(screen *ebiten.Image, world *ecs.World) {
	cx, cy, rx, ry := gameplay.TrackGeometry()
	camX, camY := cx, cy
	for _, id := range world.Entities() {
		if world.Name(id) == "Player" {
			if tr, ok := world.GetTransform(id); ok {
				camX, camY = tr.Position.X, tr.Position.Y
			}
			break
		}
	}
	sw := float64(screen.Bounds().Dx())
	sh := float64(screen.Bounds().Dy())
	scx, scy := sw/2, sh/2

	toScreen := func(wx, wy float32) (float64, float64) {
		return scx + float64(wx-camX), scy + float64(wy-camY)
	}

	// Асфальт: плотные концентрические эллипсы
	asphalt := color.RGBA{R: 55, G: 58, B: 65, A: 255}
	edge := color.RGBA{R: 220, G: 220, B: 230, A: 255}
	line := color.RGBA{R: 240, G: 200, B: 40, A: 255}
	finish := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	const segs = 96
	for t := 0; t <= 40; t++ {
		rScale := 0.55 + float32(t)*(1.0-0.55)/40
		col := asphalt
		for i := 0; i < segs; i++ {
			a0 := float64(i) * 2 * math.Pi / segs
			a1 := float64(i+1) * 2 * math.Pi / segs
			x0 := cx + rx*rScale*float32(math.Cos(a0))
			y0 := cy + ry*rScale*float32(math.Sin(a0))
			x1 := cx + rx*rScale*float32(math.Cos(a1))
			y1 := cy + ry*rScale*float32(math.Sin(a1))
			sx0, sy0 := toScreen(x0, y0)
			sx1, sy1 := toScreen(x1, y1)
			ebitenutil.DrawLine(screen, sx0, sy0, sx1, sy1, col)
		}
	}

	// Кромки и пунктир
	for i := 0; i < segs; i++ {
		a0 := float64(i) * 2 * math.Pi / segs
		a1 := float64(i+1) * 2 * math.Pi / segs
		for _, rScale := range []float32{1.0, 0.55} {
			x0 := cx + rx*rScale*float32(math.Cos(a0))
			y0 := cy + ry*rScale*float32(math.Sin(a0))
			x1 := cx + rx*rScale*float32(math.Cos(a1))
			y1 := cy + ry*rScale*float32(math.Sin(a1))
			sx0, sy0 := toScreen(x0, y0)
			sx1, sy1 := toScreen(x1, y1)
			ebitenutil.DrawLine(screen, sx0, sy0, sx1, sy1, edge)
		}
		if i%2 == 0 {
			rm := float32(0.78)
			x0 := cx + rx*rm*float32(math.Cos(a0))
			y0 := cy + ry*rm*float32(math.Sin(a0))
			x1 := cx + rx*rm*float32(math.Cos(a1))
			y1 := cy + ry*rm*float32(math.Sin(a1))
			sx0, sy0 := toScreen(x0, y0)
			sx1, sy1 := toScreen(x1, y1)
			ebitenutil.DrawLine(screen, sx0, sy0, sx1, sy1, line)
		}
	}

	// Финишная черта (низ овала)
	wps := gameplay.WaypointsForDraw()
	if len(wps) >= 2 {
		a := wps[0]
		// перпендикуляр к направлению на wp1
		bpt := wps[1]
		dx, dy := bpt.X-a.X, bpt.Y-a.Y
		lenv := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if lenv > 0.001 {
			px, py := -dy/lenv, dx/lenv
			x0, y0 := a.X+px*40, a.Y+py*40
			x1, y1 := a.X-px*40, a.Y-py*40
			sx0, sy0 := toScreen(x0, y0)
			sx1, sy1 := toScreen(x1, y1)
			ebitenutil.DrawLine(screen, sx0, sy0, sx1, sy1, finish)
			ebitenutil.DrawLine(screen, sx0+1, sy0, sx1+1, sy1, finish)
		}
	}
}

func (b *Backend) kartHUD(w *ecs.World) string {
	rs := gameplay.GetRaceState()
	phase, msg, _, _ := rs.Snapshot()

	switch phase {
	case gameplay.PhaseMenu, gameplay.PhaseCountdown, gameplay.PhaseFinished:
		return msg
	}

	var player ecs.Kart
	var rb ecs.Rigidbody
	found := false
	pos := 1
	for _, id := range w.Entities() {
		if w.Name(id) != "Player" {
			continue
		}
		player, _ = w.GetKart(id)
		rb, _ = w.GetRigidbody(id)
		pos = player.RacePosition
		found = true
		break
	}
	if !found {
		return msg
	}
	// Скорость в px/с; масштаб спрайта карта 2.5 → метры/с → км/ч
	speed := int(math.Sqrt(float64(rb.Velocity.X*rb.Velocity.X+rb.Velocity.Y*rb.Velocity.Y)) * 3.6 / 2.5)
	pu := player.PowerUp
	if pu == "" {
		pu = "-"
	} else if player.PowerUpTimer <= 0 {
		pu = pu + " [SPACE]"
	}
	out := gameplay.Tr("race.lap", itoa(player.CurrentLap+1), itoa(gameplay.LapsToWin))
	out += "  " + gameplay.Tr("race.pos", itoa(pos)) + "  " + itoa(speed) + " " + gameplay.Tr("race.kmh")
	out += "\n" + gameplay.Tr("race.time", fmtTime(rs.Elapsed()))
	if player.BestLapTime > 0 {
		out += "  " + gameplay.Tr("race.best", fmtTime(player.BestLapTime))
	}
	out += "\n" + gameplay.Tr("race.power", pu)
	out += "\n" + gameplay.Tr("race.controls")
	if msg != "" {
		out = msg + "\n" + out
	}
	return out
}

// fmtTime форматирует секунды как М:СС.д (1:23.4).
func fmtTime(sec float32) string {
	total := int(sec)
	min := total / 60
	rem := total % 60
	dec := int((sec-float32(total))*10 + 0.5)
	if dec > 9 {
		dec = 9
	}
	s := itoa(min) + ":"
	if rem < 10 {
		s += "0"
	}
	return s + itoa(rem) + "." + itoa(dec)
}

// SetUIManager sets the UI manager for this backend
func (b *Backend) SetUIManager(uiManager *ui.UIManager) {
	if b.UIContext != nil {
		b.UIContext.SetUIManager(uiManager)
	}
}

// drawGameHUD рисует графический HUD для 3D-игр (CyberNinja): полоса здоровья,
// иконки собранных предметов, экран победы/поражения. Поверх текстового DebugPrint.
func (b *Backend) drawGameHUD(screen *ebiten.Image) {
	health, maxHealth, score, total, level, status := gameplay.GameHealthState()
	if total == 0 && status == gameplay.StatusPlaying {
		return // нет игровой логики — не рисуем
	}

	sw, sh := screen.Size()
	W, H := float32(sw), float32(sh)

	// Полупрозрачные панели — рисуем через fill vector-примитивами.
	// Полоса здоровья (левый верх).
	barW := float32(240)
	barH := float32(22)
	barX := float32(16)
	barY := float32(16)
	// фон
	fillRect(screen, barX-2, barY-2, barW+4, barH+4, color.RGBA{R: 0, G: 0, B: 0, A: 140})
	// красная заливка пропорционально здоровью
	hp := float32(0)
	if maxHealth > 0 {
		hp = health / maxHealth
		if hp > 1 {
			hp = 1
		}
		if hp < 0 {
			hp = 0
		}
	}
	fillRect(screen, barX, barY, barW*hp, barH, color.RGBA{R: 220, G: 60, B: 60, A: 230})
	// рамка
	fillRect(screen, barX, barY, barW, 2, color.RGBA{R: 255, G: 255, B: 255, A: 200})
	fillRect(screen, barX, barY+barH-2, barW, 2, color.RGBA{R: 255, G: 255, B: 255, A: 200})
	fillRect(screen, barX, barY, 2, barH, color.RGBA{R: 255, G: 255, B: 255, A: 200})
	fillRect(screen, barX+barW-2, barY, 2, barH, color.RGBA{R: 255, G: 255, B: 255, A: 200})

	// Текст здоровья/счёта/уровня — используем DebugPrint в маленькие зоны.
	healthTxt := gameplay.Tr("hud.health", int(health)) + "  " + gameplay.Tr("hud.items", score, total)
	if level > 0 {
		healthTxt += "  " + gameplay.Tr("hud.level", level)
	}
	ebitenutil.DebugPrintAt(screen, healthTxt, int(barX+8), int(barY+3))

	// Экраны победы/поражения — затемнение + крупный текст по центру.
	if status == gameplay.StatusVictory || status == gameplay.StatusDefeat {
		fillRect(screen, 0, 0, W, H, color.RGBA{R: 0, G: 0, B: 0, A: 160})
		msg := gameplay.Tr("hud.defeat") + "\n" + gameplay.Tr("hud.restart")
		if status == gameplay.StatusVictory {
			msg = gameplay.Tr("hud.victory") + "\n" + gameplay.Tr("hud.next")
		}
		// Многократно печатаем по центру для «крупного» эффекта (нет TTF-шрифта в наличии).
		ebitenutil.DebugPrintAt(screen, msg, int(W/2)-120, int(H/2)-20)
	}
}

// fillRect рисует залитый прямоугольник поверх screen.
func fillRect(screen *ebiten.Image, x, y, w, h float32, col color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	rect := ebiten.NewImage(int(w), int(h))
	rect.Fill(col)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(rect, op)
}
