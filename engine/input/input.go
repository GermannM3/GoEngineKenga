package input

// Key represents a keyboard key
type Key int

// Common key constants (matching Ebiten keys)
const (
	KeyA Key = iota
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
	KeyG
	KeyH
	KeyI
	KeyJ
	KeyK
	KeyL
	KeyM
	KeyN
	KeyO
	KeyP
	KeyQ
	KeyR
	KeyS
	KeyT
	KeyU
	KeyV
	KeyW
	KeyX
	KeyY
	KeyZ
	Key0
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
	KeySpace
	KeyEnter
	KeyEscape
	KeyTab
	KeyBackspace
	KeyDelete
	KeyInsert
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	KeyShiftLeft
	KeyShiftRight
	KeyControlLeft
	KeyControlRight
	KeyAltLeft
	KeyAltRight
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyMax
)

// MouseButton represents a mouse button
type MouseButton int

const (
	MouseButtonLeft MouseButton = iota
	MouseButtonMiddle
	MouseButtonRight
	MouseButtonMax
)

// PadButton — кнопка геймпада в стандартной раскладке (XBox/PS).
// Порядок совпадает с GLFW/XInput; для ebiten есть таблица соответствия.
type PadButton int

const (
	PadA PadButton = iota // нижняя (A / Cross)
	PadB                  // правая (B / Circle)
	PadX                  // левая (X / Square)
	PadY                  // верхняя (Y / Triangle)
	PadLB
	PadRB
	PadBack
	PadStart
	PadGuide // центральная (Xbox/PS) — в XInput не приходит
	PadL3
	PadR3
	PadUp
	PadDown
	PadLeft
	PadRight

	PadButtonMax
)

// PadAxis — ось геймпада: стики -1..1, триггеры 0..1.
// Стики: LY/RY положительны ВНИЗ (стандартная конвенция).
type PadAxis int

const (
	PadAxisLX PadAxis = iota
	PadAxisLY
	PadAxisRX
	PadAxisRY
	PadAxisLT
	PadAxisRT

	PadAxisMax
)

// Deadzone обнуляет малые отклонения оси (дрейф аналоговых стиков).
func Deadzone(v, deadzone float32) float32 {
	if v < deadzone && v > -deadzone {
		return 0
	}
	return v
}

// State holds the current input state
type State struct {
	// Keyboard state
	keysPressed     [KeyMax]bool
	keysPressedPrev [KeyMax]bool

	// Mouse state
	MouseX, MouseY             int
	MouseDeltaX, MouseDeltaY   int
	prevMouseX, prevMouseY     int
	mouseButtons               [MouseButtonMax]bool
	mouseButtonsPrev           [MouseButtonMax]bool
	MouseScrollX, MouseScrollY float64

	// Gamepad state (заполняется бэкендом: XInput на Windows/webgpu,
	// встроенный API ebiten в ebiten-бэкенде)
	padConnected   bool
	padButtons     [PadButtonMax]bool
	padButtonsPrev [PadButtonMax]bool
	padAxes        [PadAxisMax]float32
}

// NewState creates a new input state
func NewState() *State {
	return &State{}
}

// Update should be called at the start of each frame to update deltas
func (s *State) Update() {
	// Calculate mouse delta
	s.MouseDeltaX = s.MouseX - s.prevMouseX
	s.MouseDeltaY = s.MouseY - s.prevMouseY
	s.prevMouseX = s.MouseX
	s.prevMouseY = s.MouseY
}

// EndFrame should be called at the end of each frame to store previous state
func (s *State) EndFrame() {
	copy(s.keysPressedPrev[:], s.keysPressed[:])
	copy(s.mouseButtonsPrev[:], s.mouseButtons[:])
	copy(s.padButtonsPrev[:], s.padButtons[:])
	s.MouseScrollX = 0
	s.MouseScrollY = 0
}

// ---- Геймпад ----

// SetPadConnected задаёт факт подключения геймпада (вызывается бэкендом).
func (s *State) SetPadConnected(connected bool) { s.padConnected = connected }

// IsPadConnected сообщает, есть ли геймпад.
func (s *State) IsPadConnected() bool { return s.padConnected }

// SetPadButton задаёт состояние кнопки геймпада (вызывается бэкендом).
func (s *State) SetPadButton(btn PadButton, pressed bool) {
	if btn >= 0 && btn < PadButtonMax {
		s.padButtons[btn] = pressed
	}
}

// IsPadButtonDown — кнопка удерживается.
func (s *State) IsPadButtonDown(btn PadButton) bool {
	if btn >= 0 && btn < PadButtonMax {
		return s.padButtons[btn]
	}
	return false
}

// IsPadButtonJustPressed — кнопка нажата в этом кадре.
func (s *State) IsPadButtonJustPressed(btn PadButton) bool {
	if btn >= 0 && btn < PadButtonMax {
		return s.padButtons[btn] && !s.padButtonsPrev[btn]
	}
	return false
}

// IsPadButtonJustReleased — кнопка отпущена в этом кадре.
func (s *State) IsPadButtonJustReleased(btn PadButton) bool {
	if btn >= 0 && btn < PadButtonMax {
		return !s.padButtons[btn] && s.padButtonsPrev[btn]
	}
	return false
}

// SetPadAxis задаёт значение оси (вызывается бэкендом).
func (s *State) SetPadAxis(axis PadAxis, v float32) {
	if axis >= 0 && axis < PadAxisMax {
		s.padAxes[axis] = v
	}
}

// PadAxisValue возвращает значение оси, -1..1 (триггеры 0..1).
func (s *State) PadAxisValue(axis PadAxis) float32 {
	if axis >= 0 && axis < PadAxisMax {
		return s.padAxes[axis]
	}
	return 0
}

// SetKeyPressed sets a key's pressed state
func (s *State) SetKeyPressed(key Key, pressed bool) {
	if key >= 0 && key < KeyMax {
		s.keysPressed[key] = pressed
	}
}

// SetMouseButton sets a mouse button's pressed state
func (s *State) SetMouseButton(btn MouseButton, pressed bool) {
	if btn >= 0 && btn < MouseButtonMax {
		s.mouseButtons[btn] = pressed
	}
}

// SetMousePosition sets the mouse position
func (s *State) SetMousePosition(x, y int) {
	s.MouseX = x
	s.MouseY = y
}

// SetMouseScroll sets the mouse scroll values
func (s *State) SetMouseScroll(x, y float64) {
	s.MouseScrollX = x
	s.MouseScrollY = y
}

// IsKeyPressed returns true if the key is currently pressed
func (s *State) IsKeyPressed(key Key) bool {
	if key >= 0 && key < KeyMax {
		return s.keysPressed[key]
	}
	return false
}

// IsKeyJustPressed returns true if the key was just pressed this frame
func (s *State) IsKeyJustPressed(key Key) bool {
	if key >= 0 && key < KeyMax {
		return s.keysPressed[key] && !s.keysPressedPrev[key]
	}
	return false
}

// IsKeyJustReleased returns true if the key was just released this frame
func (s *State) IsKeyJustReleased(key Key) bool {
	if key >= 0 && key < KeyMax {
		return !s.keysPressed[key] && s.keysPressedPrev[key]
	}
	return false
}

// IsMouseButtonPressed returns true if the mouse button is currently pressed
func (s *State) IsMouseButtonPressed(btn MouseButton) bool {
	if btn >= 0 && btn < MouseButtonMax {
		return s.mouseButtons[btn]
	}
	return false
}

// IsMouseButtonJustPressed returns true if the mouse button was just pressed
func (s *State) IsMouseButtonJustPressed(btn MouseButton) bool {
	if btn >= 0 && btn < MouseButtonMax {
		return s.mouseButtons[btn] && !s.mouseButtonsPrev[btn]
	}
	return false
}

// IsMouseButtonJustReleased returns true if the mouse button was just released
func (s *State) IsMouseButtonJustReleased(btn MouseButton) bool {
	if btn >= 0 && btn < MouseButtonMax {
		return !s.mouseButtons[btn] && s.mouseButtonsPrev[btn]
	}
	return false
}

// GetPressedKeys returns a slice of all currently pressed keys
func (s *State) GetPressedKeys() []Key {
	var keys []Key
	for k := Key(0); k < KeyMax; k++ {
		if s.keysPressed[k] {
			keys = append(keys, k)
		}
	}
	return keys
}

// Reset clears all input state
func (s *State) Reset() {
	for i := range s.keysPressed {
		s.keysPressed[i] = false
		s.keysPressedPrev[i] = false
	}
	for i := range s.mouseButtons {
		s.mouseButtons[i] = false
		s.mouseButtonsPrev[i] = false
	}
	s.MouseX, s.MouseY = 0, 0
	s.MouseDeltaX, s.MouseDeltaY = 0, 0
	s.prevMouseX, s.prevMouseY = 0, 0
	s.MouseScrollX, s.MouseScrollY = 0, 0
}
