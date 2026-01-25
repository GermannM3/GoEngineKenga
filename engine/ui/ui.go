package ui

import (
	"image/color"

	emath "goenginekenga/engine/math"
)

// UIElement базовый интерфейс для UI элементов
type UIElement interface {
	Update(deltaTime float32)
	Render(canvas *Canvas)
	SetPosition(pos emath.Vec2)
	GetPosition() emath.Vec2
	SetSize(size emath.Vec2)
	GetSize() emath.Vec2
	SetVisible(visible bool)
	IsVisible() bool
}

// Canvas представляет поверхность для рисования UI
type Canvas struct {
	Width  int
	Height int
	// В будущем здесь будет буфер для рендеринга
}

// NewCanvas создает новый canvas
func NewCanvas(width, height int) *Canvas {
	return &Canvas{
		Width:  width,
		Height: height,
	}
}

// Button представляет интерактивную кнопку
type Button struct {
	Text     string
	Position emath.Vec2
	Size     emath.Vec2
	Visible  bool

	OnClick func()

	color      color.RGBA
	hoverColor color.RGBA
	isHovered  bool
	isPressed  bool
}

func NewButton(text string, position, size emath.Vec2) *Button {
	return &Button{
		Text:       text,
		Position:   position,
		Size:       size,
		Visible:    true,
		color:      color.RGBA{100, 150, 200, 255},
		hoverColor: color.RGBA{120, 170, 220, 255},
		isHovered:  false,
		isPressed:  false,
	}
}

func (b *Button) Update(deltaTime float32) {
	// Обновление состояния кнопки
}

func (b *Button) Render(canvas *Canvas) {
	if !b.Visible {
		return
	}

	// Простой рендеринг кнопки (в будущем через canvas API)
	currentColor := b.color
	if b.isHovered {
		currentColor = b.hoverColor
	}

	_ = currentColor // Пока не рисуем, но цвет определен
}

func (b *Button) SetPosition(pos emath.Vec2) { b.Position = pos }
func (b *Button) GetPosition() emath.Vec2    { return b.Position }
func (b *Button) SetSize(size emath.Vec2)    { b.Size = size }
func (b *Button) GetSize() emath.Vec2        { return b.Size }
func (b *Button) SetVisible(visible bool)    { b.Visible = visible }
func (b *Button) IsVisible() bool            { return b.Visible }

// Label представляет текстовую метку
type Label struct {
	Text     string
	Position emath.Vec2
	Size     emath.Vec2
	Visible  bool

	Color    color.RGBA
	FontSize float32
}

func NewLabel(text string, position emath.Vec2) *Label {
	return &Label{
		Text:     text,
		Position: position,
		Size:     emath.V2(100, 20),
		Visible:  true,
		Color:    color.RGBA{255, 255, 255, 255},
		FontSize: 12,
	}
}

func (l *Label) Update(deltaTime float32) {
	// Ничего не делаем
}

func (l *Label) Render(canvas *Canvas) {
	if !l.Visible {
		return
	}

	// Простой рендеринг текста (в будущем через canvas API)
	_ = l.Color
}

func (l *Label) SetPosition(pos emath.Vec2) { l.Position = pos }
func (l *Label) GetPosition() emath.Vec2    { return l.Position }
func (l *Label) SetSize(size emath.Vec2)    { l.Size = size }
func (l *Label) GetSize() emath.Vec2        { return l.Size }
func (l *Label) SetVisible(visible bool)    { l.Visible = visible }
func (l *Label) IsVisible() bool            { return l.Visible }

// Panel представляет контейнер для других элементов
type Panel struct {
	Position emath.Vec2
	Size     emath.Vec2
	Visible  bool

	BackgroundColor color.RGBA
	Children        []UIElement
}

func NewPanel(position, size emath.Vec2) *Panel {
	return &Panel{
		Position:        position,
		Size:            size,
		Visible:         true,
		BackgroundColor: color.RGBA{50, 50, 50, 200},
		Children:        make([]UIElement, 0),
	}
}

func (p *Panel) AddChild(child UIElement) {
	p.Children = append(p.Children, child)
}

func (p *Panel) Update(deltaTime float32) {
	if !p.Visible {
		return
	}

	for _, child := range p.Children {
		child.Update(deltaTime)
	}
}

func (p *Panel) Render(canvas *Canvas) {
	if !p.Visible {
		return
	}

	// Рендеринг фона панели
	_ = p.BackgroundColor

	// Рендеринг дочерних элементов
	for _, child := range p.Children {
		child.Render(canvas)
	}
}

func (p *Panel) SetPosition(pos emath.Vec2) { p.Position = pos }
func (p *Panel) GetPosition() emath.Vec2    { return p.Position }
func (p *Panel) SetSize(size emath.Vec2)    { p.Size = size }
func (p *Panel) GetSize() emath.Vec2        { return p.Size }
func (p *Panel) SetVisible(visible bool)    { p.Visible = visible }
func (p *Panel) IsVisible() bool            { return p.Visible }

// UIManager управляет UI системой
type UIManager struct {
	Canvas   *Canvas
	Elements []UIElement

	// Состояние ввода
	MouseX, MouseY float32
	MousePressed   bool
}

// NewUIManager создает новый UI менеджер
func NewUIManager(canvas *Canvas) *UIManager {
	return &UIManager{
		Canvas:   canvas,
		Elements: make([]UIElement, 0),
	}
}

// AddElement добавляет UI элемент
func (ui *UIManager) AddElement(element UIElement) {
	ui.Elements = append(ui.Elements, element)
}

// Update обновляет все UI элементы
func (ui *UIManager) Update(deltaTime float32) {
	for _, element := range ui.Elements {
		element.Update(deltaTime)
	}
}

// Render рендерит все UI элементы
func (ui *UIManager) Render() {
	for _, element := range ui.Elements {
		element.Render(ui.Canvas)
	}
}

// HandleInput обрабатывает ввод для UI
func (ui *UIManager) HandleInput(mouseX, mouseY float32, mousePressed bool) {
	ui.MouseX = mouseX
	ui.MouseY = mouseY
	ui.MousePressed = mousePressed

	// Проверяем взаимодействие с элементами
	for _, element := range ui.Elements {
		if button, ok := element.(*Button); ok {
			// Проверка попадания курсора в кнопку
			if mouseX >= button.Position.X && mouseX <= button.Position.X+button.Size.X &&
				mouseY >= button.Position.Y && mouseY <= button.Position.Y+button.Size.Y {
				button.isHovered = true

				if mousePressed && button.OnClick != nil {
					button.OnClick()
				}
			} else {
				button.isHovered = false
			}
		}
	}
}
