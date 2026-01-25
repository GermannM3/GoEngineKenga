package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	emath "goenginekenga/engine/math"
)

// EbitenRenderer renders UI elements using Ebiten
type EbitenRenderer struct {
	screen *ebiten.Image
}

// NewEbitenRenderer creates a new Ebiten UI renderer
func NewEbitenRenderer() *EbitenRenderer {
	return &EbitenRenderer{}
}

// SetScreen sets the target screen for rendering
func (r *EbitenRenderer) SetScreen(screen *ebiten.Image) {
	r.screen = screen
}

// DrawRect draws a filled rectangle
func (r *EbitenRenderer) DrawRect(x, y, w, h float32, col color.RGBA) {
	if r.screen == nil {
		return
	}
	vector.DrawFilledRect(r.screen, x, y, w, h, col, false)
}

// DrawRectOutline draws a rectangle outline
func (r *EbitenRenderer) DrawRectOutline(x, y, w, h float32, col color.RGBA, thickness float32) {
	if r.screen == nil {
		return
	}
	// Top
	vector.DrawFilledRect(r.screen, x, y, w, thickness, col, false)
	// Bottom
	vector.DrawFilledRect(r.screen, x, y+h-thickness, w, thickness, col, false)
	// Left
	vector.DrawFilledRect(r.screen, x, y, thickness, h, col, false)
	// Right
	vector.DrawFilledRect(r.screen, x+w-thickness, y, thickness, h, col, false)
}

// DrawText draws text at the specified position
// Note: Uses Ebiten's debug font for simplicity
func (r *EbitenRenderer) DrawText(text string, x, y float32, col color.RGBA) {
	if r.screen == nil || text == "" {
		return
	}
	// Use ebitenutil.DebugPrintAt for simple text rendering
	// For production, use ebiten/v2/text with custom fonts
	ebitenutil.DebugPrintAt(r.screen, text, int(x), int(y))
}

// DrawTextCentered draws text centered at the specified position
func (r *EbitenRenderer) DrawTextCentered(text string, x, y, w, h float32, col color.RGBA) {
	if r.screen == nil || text == "" {
		return
	}
	// Approximate character width (debug font is ~6 pixels wide per char)
	charWidth := float32(6)
	charHeight := float32(16)

	textWidth := float32(len(text)) * charWidth
	textX := x + (w-textWidth)/2
	textY := y + (h-charHeight)/2

	ebitenutil.DebugPrintAt(r.screen, text, int(textX), int(textY))
}

// DrawCircle draws a filled circle
func (r *EbitenRenderer) DrawCircle(cx, cy, radius float32, col color.RGBA) {
	if r.screen == nil {
		return
	}
	vector.DrawFilledCircle(r.screen, cx, cy, radius, col, false)
}

// DrawLine draws a line between two points
func (r *EbitenRenderer) DrawLine(x1, y1, x2, y2 float32, col color.RGBA, thickness float32) {
	if r.screen == nil {
		return
	}
	vector.StrokeLine(r.screen, x1, y1, x2, y2, thickness, col, false)
}

// DrawImage draws an image at the specified position
func (r *EbitenRenderer) DrawImage(img *ebiten.Image, x, y, w, h float32) {
	if r.screen == nil || img == nil {
		return
	}

	imgW, imgH := img.Bounds().Dx(), img.Bounds().Dy()
	scaleX := w / float32(imgW)
	scaleY := h / float32(imgH)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(scaleX), float64(scaleY))
	op.GeoM.Translate(float64(x), float64(y))

	r.screen.DrawImage(img, op)
}

// RenderUIManager renders all UI elements from a UIManager
func (r *EbitenRenderer) RenderUIManager(ui *UIManager) {
	if r.screen == nil || ui == nil {
		return
	}

	for _, element := range ui.Elements {
		r.RenderElement(element)
	}
}

// RenderElement renders a single UI element
func (r *EbitenRenderer) RenderElement(element UIElement) {
	if element == nil || !element.IsVisible() {
		return
	}

	switch e := element.(type) {
	case *Button:
		r.renderButton(e)
	case *Label:
		r.renderLabel(e)
	case *Panel:
		r.renderPanel(e)
	}
}

// renderButton renders a button
func (r *EbitenRenderer) renderButton(b *Button) {
	if !b.Visible {
		return
	}

	// Background color (changes on hover)
	bgColor := b.color
	if b.isHovered {
		bgColor = b.hoverColor
	}
	if b.isPressed {
		// Darken when pressed
		bgColor = color.RGBA{
			R: uint8(float32(bgColor.R) * 0.8),
			G: uint8(float32(bgColor.G) * 0.8),
			B: uint8(float32(bgColor.B) * 0.8),
			A: bgColor.A,
		}
	}

	// Draw button background
	r.DrawRect(b.Position.X, b.Position.Y, b.Size.X, b.Size.Y, bgColor)

	// Draw button outline
	r.DrawRectOutline(b.Position.X, b.Position.Y, b.Size.X, b.Size.Y,
		color.RGBA{R: 255, G: 255, B: 255, A: 100}, 1)

	// Draw button text (centered)
	r.DrawTextCentered(b.Text, b.Position.X, b.Position.Y, b.Size.X, b.Size.Y,
		color.RGBA{R: 255, G: 255, B: 255, A: 255})
}

// renderLabel renders a label
func (r *EbitenRenderer) renderLabel(l *Label) {
	if !l.Visible {
		return
	}

	r.DrawText(l.Text, l.Position.X, l.Position.Y, l.Color)
}

// renderPanel renders a panel with its children
func (r *EbitenRenderer) renderPanel(p *Panel) {
	if !p.Visible {
		return
	}

	// Draw panel background
	r.DrawRect(p.Position.X, p.Position.Y, p.Size.X, p.Size.Y, p.BackgroundColor)

	// Render children
	for _, child := range p.Children {
		r.RenderElement(child)
	}
}

// UIRenderContext provides context for UI rendering within game loop
type UIRenderContext struct {
	Renderer   *EbitenRenderer
	UIManager  *UIManager
	InputState struct {
		MouseX, MouseY int
		MousePressed   bool
	}
}

// NewUIRenderContext creates a new UI render context
func NewUIRenderContext() *UIRenderContext {
	return &UIRenderContext{
		Renderer:  NewEbitenRenderer(),
		UIManager: nil,
	}
}

// SetUIManager sets the UI manager for this context
func (ctx *UIRenderContext) SetUIManager(ui *UIManager) {
	ctx.UIManager = ui
}

// Update updates the UI with input
func (ctx *UIRenderContext) Update(mouseX, mouseY int, mousePressed bool) {
	ctx.InputState.MouseX = mouseX
	ctx.InputState.MouseY = mouseY
	ctx.InputState.MousePressed = mousePressed

	if ctx.UIManager != nil {
		ctx.UIManager.HandleInput(float32(mouseX), float32(mouseY), mousePressed)
	}
}

// Render renders the UI to the screen
func (ctx *UIRenderContext) Render(screen *ebiten.Image) {
	ctx.Renderer.SetScreen(screen)

	if ctx.UIManager != nil {
		ctx.Renderer.RenderUIManager(ctx.UIManager)
	}
}

// Helper to convert Vec2 to screen coordinates (if needed)
func vec2ToScreen(v emath.Vec2, screenWidth, screenHeight int) (float32, float32) {
	return v.X, v.Y
}
