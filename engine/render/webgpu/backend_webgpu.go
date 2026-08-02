//go:build webgpu && !js

package webgpu

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/cogentcore/webgpu/wgpu"
	"github.com/cogentcore/webgpu/wgpuglfw"
	"github.com/go-gl/glfw/v3.3/glfw"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/render"
)

// Backend (webgpu) — GPU рендер 3D-сцены через WebGPU.
type Backend struct {
	title  string
	width  int
	height int

	// Orbit camera (ПКМ rotate, СКМ pan, scroll zoom) — как в ebiten backend
	orbitState   render.OrbitState
	orbitEnabled bool
	orbitSynced  bool

	// Mouse state
	lastMouseX, lastMouseY float64
	mouseDX, mouseDY       float64
	rightDown, middleDown  bool
	scrollDelta            float64
}

func init() {
	runtime.LockOSThread()
}

func New(title string, width, height int) *Backend {
	return &Backend{title: title, width: width, height: height}
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

	window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
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

		if initial != nil && initial.World != nil && b.orbitEnabled {
			b.updateOrbitCamera(initial)
		}
		b.scrollDelta = 0

		if initial != nil && initial.OnUpdate != nil {
			initial.OnUpdate(dt)
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
	}
	return nil
}

// updateOrbitCamera применяет orbit/pan/zoom к первой 3D-камере сцены
// (зеркало ebiten/backend.go:updateOrbitCamera).
func (b *Backend) updateOrbitCamera(frame *render.Frame) {
	w := frame.World
	var camID ecs.EntityID
	var hasCam bool
	for _, id := range w.Entities() {
		if _, ok := w.GetCamera(id); ok {
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
