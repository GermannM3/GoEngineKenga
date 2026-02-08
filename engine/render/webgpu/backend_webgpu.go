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
	"goenginekenga/engine/render"
)

// Backend (webgpu) — GPU рендер 3D-сцены через WebGPU.
type Backend struct {
	title  string
	width  int
	height int
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

	lastTime := time.Now()
	for !window.ShouldClose() {
		glfw.PollEvents()

		dt := time.Since(lastTime).Seconds()
		lastTime = time.Now()
		if dt > 0.1 {
			dt = 1.0 / 60.0
		}

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
