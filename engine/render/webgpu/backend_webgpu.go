//go:build webgpu && !js

package webgpu

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/cogentcore/webgpu/wgpuglfw"
	"github.com/go-gl/glfw/v3.3/glfw"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/ecs"
	"goenginekenga/engine/input"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/render"
)

// Backend (webgpu) — GPU рендер 3D-сцены через WebGPU.
type Backend struct {
	title      string
	width      int
	height     int
	InputState *input.State // заполняется каждый кадр (клавиатура/мышь), как в ebiten backend

	// Orbit camera (ПКМ rotate, СКМ pan, scroll zoom) — как в ebiten backend
	orbitState   render.OrbitState
	orbitEnabled bool
	orbitSynced  bool

	// Mouse state
	lastMouseX, lastMouseY float64
	mouseDX, mouseDY       float64
	rightDown, middleDown  bool
	scrollDelta            float64

	lastHUD string // последний показанный HUDText (чтобы не дёргать заголовок)
}

func init() {
	runtime.LockOSThread()
}

func New(title string, width, height int) *Backend {
	return &Backend{title: title, width: width, height: height, InputState: input.NewState()}
}

func (b *Backend) RunLoop(initial *render.Frame) error {
	if err := glfw.Init(); err != nil {
		return err
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(b.width, b.height, b.title, nil, nil)
	if err != nil {
		return err
	}
	defer window.Destroy()

	s, err := initState(window, wgpuglfw.GetSurfaceDescriptor(window))
	if err != nil {
		return err
	}
	defer s.Destroy()

	var resolver *asset.Resolver
	if initial != nil && initial.ProjectDir != "" {
		if r, ok := initial.Resolver.(*asset.Resolver); ok && r != nil {
			resolver = r
		} else if r, err := asset.NewResolver(initial.ProjectDir); err == nil {
			resolver = r
		}
	}

	// Ввод: клавиатура через колбэк, мышь опрашивается каждый кадр.
	// initial.InputState нужен run.go, чтобы игровые системы получили состояние.
	initial.InputState = b.InputState

	window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if k, ok := glfwKeyToInput(key); ok {
			b.InputState.SetKeyPressed(k, action == glfw.Press || action == glfw.Repeat)
		}
		if key == glfw.KeyR && (action == glfw.Press || action == glfw.Repeat) {
			report := s.instance.GenerateReport()
			buf, _ := json.MarshalIndent(report, "", "  ")
			fmt.Print(string(buf))
		}
	})

	window.SetSizeCallback(func(_ *glfw.Window, width, height int) {
		s.Resize(width, height)
	})

	// Orbit camera: ПКМ rotate, СКМ pan, scroll zoom
	b.orbitState = render.DefaultOrbitState()
	b.orbitEnabled = true
	window.SetMouseButtonCallback(func(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		switch button {
		case glfw.MouseButtonRight:
			b.rightDown = action == glfw.Press
		case glfw.MouseButtonMiddle:
			b.middleDown = action == glfw.Press
		}
	})
	window.SetScrollCallback(func(_ *glfw.Window, xoff, yoff float64) {
		b.scrollDelta += yoff
	})

	lastTime := time.Now()
	for !window.ShouldClose() {
		glfw.PollEvents()

		// Deltas мыши между кадрами
		mx, my := window.GetCursorPos()
		b.mouseDX = mx - b.lastMouseX
		b.mouseDY = my - b.lastMouseY
		b.lastMouseX, b.lastMouseY = mx, my

		dt := time.Since(lastTime).Seconds()
		lastTime = time.Now()
		if dt > 0.1 {
			dt = 1.0 / 60.0
		}

		// Мышь: позиция и кнопки в input.State (орбита ниже использует свои raw-флаги)
		mx, my := window.GetCursorPos()
		b.InputState.SetMousePosition(int(mx), int(my))
		b.InputState.SetMouseButton(input.MouseButtonLeft, window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press)
		b.InputState.SetMouseButton(input.MouseButtonMiddle, window.GetMouseButton(glfw.MouseButtonMiddle) == glfw.Press)
		b.InputState.SetMouseButton(input.MouseButtonRight, window.GetMouseButton(glfw.MouseButtonRight) == glfw.Press)
		b.InputState.Update()

		if initial != nil && initial.World != nil && b.orbitEnabled {
			b.updateOrbitCamera(initial)
		}
		b.scrollDelta = 0

		if initial != nil && initial.OnUpdate != nil {
			initial.OnUpdate(dt)
		}

		// HUD без 2D-оверлея — в заголовке окна (обновляем только при изменении)
		if initial != nil && initial.HUDText != b.lastHUD {
			b.lastHUD = initial.HUDText
			title := b.title
			if b.lastHUD != "" {
				title += "  —  " + strings.ReplaceAll(b.lastHUD, "\n", " | ")
			}
			window.SetTitle(title)
		}

		if err := s.RenderScene(initial, resolver); err != nil {
			errstr := err.Error()
			switch {
			case strings.Contains(errstr, "Surface timed out"):
			case strings.Contains(errstr, "Surface is outdated"):
			case strings.Contains(errstr, "Surface was lost"):
			default:
				return err
			}
		}
		b.InputState.EndFrame()
	}
	return nil
}

// glfwKeyToInput маппит клавишу glfw в engine/input.Key (порядок констант совпадает с ebiten).
func glfwKeyToInput(k glfw.Key) (input.Key, bool) {
	switch {
	case k >= glfw.KeyA && k <= glfw.KeyZ:
		return input.KeyA + input.Key(k-glfw.KeyA), true
	case k >= glfw.Key0 && k <= glfw.Key9:
		return input.Key0 + input.Key(k-glfw.Key0), true
	}
	switch k {
	case glfw.KeySpace:
		return input.KeySpace, true
	case glfw.KeyEnter:
		return input.KeyEnter, true
	case glfw.KeyEscape:
		return input.KeyEscape, true
	case glfw.KeyTab:
		return input.KeyTab, true
	case glfw.KeyBackspace:
		return input.KeyBackspace, true
	case glfw.KeyDelete:
		return input.KeyDelete, true
	case glfw.KeyInsert:
		return input.KeyInsert, true
	case glfw.KeyHome:
		return input.KeyHome, true
	case glfw.KeyEnd:
		return input.KeyEnd, true
	case glfw.KeyPageUp:
		return input.KeyPageUp, true
	case glfw.KeyPageDown:
		return input.KeyPageDown, true
	case glfw.KeyUp:
		return input.KeyArrowUp, true
	case glfw.KeyDown:
		return input.KeyArrowDown, true
	case glfw.KeyLeft:
		return input.KeyArrowLeft, true
	case glfw.KeyRight:
		return input.KeyArrowRight, true
	case glfw.KeyLeftShift:
		return input.KeyShiftLeft, true
	case glfw.KeyRightShift:
		return input.KeyShiftRight, true
	case glfw.KeyLeftControl:
		return input.KeyControlLeft, true
	case glfw.KeyRightControl:
		return input.KeyControlRight, true
	case glfw.KeyLeftAlt:
		return input.KeyAltLeft, true
	case glfw.KeyRightAlt:
		return input.KeyAltRight, true
	case glfw.KeyF1:
		return input.KeyF1, true
	case glfw.KeyF2:
		return input.KeyF2, true
	case glfw.KeyF3:
		return input.KeyF3, true
	case glfw.KeyF4:
		return input.KeyF4, true
	case glfw.KeyF5:
		return input.KeyF5, true
	case glfw.KeyF6:
		return input.KeyF6, true
	case glfw.KeyF7:
		return input.KeyF7, true
	case glfw.KeyF8:
		return input.KeyF8, true
	case glfw.KeyF9:
		return input.KeyF9, true
	case glfw.KeyF10:
		return input.KeyF10, true
	case glfw.KeyF11:
		return input.KeyF11, true
	case glfw.KeyF12:
		return input.KeyF12, true
	}
	return 0, false
}

// updateOrbitCamera применяет orbit/pan/zoom к первой 3D-камере сцены
// (зеркало ebiten/backend.go:updateOrbitCamera). Следящие камеры (FollowID != 0)
// не трогаются — ими управляет runtime.UpdateFollowCameras.
func (b *Backend) updateOrbitCamera(frame *render.Frame) {
	w := frame.World
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
	if frame.OrbitResetRequested {
		frame.OrbitResetRequested = false
		b.orbitSynced = false
	}
	if !b.orbitSynced {
		b.orbitState.SyncFromTransform(tr.Position, tr.Rotation.Y, tr.Rotation.X)
		b.orbitSynced = true
	}

	// ПКМ: orbit
	if b.rightDown {
		b.orbitState.Orbit(float32(b.mouseDX), float32(b.mouseDY))
	}
	// СКМ: pan
	if b.middleDown {
		b.orbitState.Pan(float32(b.mouseDX), float32(b.mouseDY))
	}
	// Scroll: zoom
	if b.scrollDelta != 0 {
		b.orbitState.Zoom(float32(b.scrollDelta))
	}

	pos := b.orbitState.Position()
	tr.Position = pos
	tr.Rotation = emath.Vec3{X: b.orbitState.Pitch, Y: b.orbitState.Yaw, Z: tr.Rotation.Z}
	w.SetTransform(camID, tr)
}
