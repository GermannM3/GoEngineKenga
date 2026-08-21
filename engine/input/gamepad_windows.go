//go:build windows

package input

import (
	"sync"
	"syscall"
	"unsafe"
)

// Геймпад на Windows через XInput (syscall, без cgo): работает и в
// webgpu-бэкенде, где go-gl/glfw не экспортирует gamepad API.

type xinputGamepad struct {
	buttons                   uint16
	leftTrigger, rightTrigger uint8
	lx, ly, rx, ry            int16
}

type xinputState struct {
	packet  uint32
	gamepad xinputGamepad
}

const (
	xinputErrSuccess = 0

	// Маски XINPUT_GAMEPAD
	xiDPadUp    = 0x0001
	xiDPadDown  = 0x0002
	xiDPadLeft  = 0x0004
	xiDPadRight = 0x0008
	xiStart     = 0x0010
	xiBack      = 0x0020
	xiL3        = 0x0040
	xiR3        = 0x0080
	xiLB        = 0x0100
	xiRB        = 0x0200
	xiA         = 0x1000
	xiB         = 0x2000
	xiX         = 0x4000
	xiY         = 0x8000
)

var (
	xinputOnce sync.Once
	xinputGet  *syscall.LazyProc // nil — XInput нет в системе
)

func loadXInput() {
	for _, name := range []string{"xinput1_4.dll", "xinput1_3.dll", "xinput9_1_0.dll"} {
		dll := syscall.NewLazyDLL(name)
		p := dll.NewProc("XInputGetState")
		if err := dll.Load(); err == nil {
			if err := p.Find(); err == nil {
				xinputGet = p
				return
			}
		}
	}
}

// PollNativeGamepad читает первый XInput-геймпад в состояние ввода.
// Вызывается бэкендом каждый кадр (webgpu; ebiten опрашивает сам).
func (s *State) PollNativeGamepad() {
	xinputOnce.Do(loadXInput)
	if xinputGet == nil {
		s.SetPadConnected(false)
		return
	}
	var st xinputState
	r, _, _ := xinputGet.Call(0, uintptr(unsafe.Pointer(&st)))
	if r != uintptr(xinputErrSuccess) {
		s.SetPadConnected(false)
		return
	}
	s.SetPadConnected(true)

	b := st.gamepad.buttons
	set := func(mask uint16, btn PadButton) { s.SetPadButton(btn, b&mask != 0) }
	set(xiA, PadA)
	set(xiB, PadB)
	set(xiX, PadX)
	set(xiY, PadY)
	set(xiLB, PadLB)
	set(xiRB, PadRB)
	set(xiBack, PadBack)
	set(xiStart, PadStart)
	set(xiL3, PadL3)
	set(xiR3, PadR3)
	set(xiDPadUp, PadUp)
	set(xiDPadDown, PadDown)
	set(xiDPadLeft, PadLeft)
	set(xiDPadRight, PadRight)

	const maxS16 = 32767.0
	g := &st.gamepad
	s.SetPadAxis(PadAxisLX, float32(g.lx)/maxS16)
	s.SetPadAxis(PadAxisLY, -float32(g.ly)/maxS16) // XInput: вверх положителен → стандарт «вниз +»
	s.SetPadAxis(PadAxisRX, float32(g.rx)/maxS16)
	s.SetPadAxis(PadAxisRY, -float32(g.ry)/maxS16)
	s.SetPadAxis(PadAxisLT, float32(g.leftTrigger)/255)
	s.SetPadAxis(PadAxisRT, float32(g.rightTrigger)/255)
}
