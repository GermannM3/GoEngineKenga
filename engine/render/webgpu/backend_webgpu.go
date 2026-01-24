//go:build webgpu && !js

package webgpu

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"github.com/cogentcore/webgpu/wgpu"
	"github.com/cogentcore/webgpu/wgpuglfw"
	"github.com/go-gl/glfw/v3.3/glfw"

	"goenginekenga/engine/render"
)

// Backend (webgpu) — минимальная реализация на базе cogentcore/webgpu.
// v0: рисуем треугольник, чтобы подтвердить «цепочку» окно→surface→pipeline→present.
type Backend struct {
	title string
	width int
	height int
}

func init() {
	runtime.LockOSThread()
}

func New(title string, width, height int) *Backend {
	return &Backend{title: title, width: width, height: height}
}

func (b *Backend) RunLoop(initial *render.Frame) error {
	_ = initial

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

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		// Print resource usage on pressing 'R'
		if key == glfw.KeyR && (action == glfw.Press || action == glfw.Repeat) {
			report := s.instance.GenerateReport()
			buf, _ := json.MarshalIndent(report, "", "  ")
			fmt.Print(string(buf))
		}
	})

	window.SetSizeCallback(func(_ *glfw.Window, width, height int) {
		s.Resize(width, height)
	})

	for !window.ShouldClose() {
		glfw.PollEvents()
		if err := s.Render(); err != nil {
			errstr := err.Error()
			switch {
			case strings.Contains(errstr, "Surface timed out"):
				// ignore
			case strings.Contains(errstr, "Surface is outdated"):
				// ignore
			case strings.Contains(errstr, "Surface was lost"):
				// ignore
			default:
				return err
			}
		}
	}
	return nil
}

