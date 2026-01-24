package cli

import (
	"context"
	"image/color"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"goenginekenga/engine/render"
	"goenginekenga/engine/render/ebiten"
	"goenginekenga/engine/render/webgpu"
	"goenginekenga/engine/runtime"
	"goenginekenga/engine/scene"
	"goenginekenga/engine/script"
)

func newRunCommand() *cobra.Command {
	var projectDir string
	var scenePath string
	var backend string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a project (runtime window)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var s *scene.Scene
			if scenePath == "" {
				s = scene.DefaultScene()
			} else {
				sp := filepath.Join(projectDir, scenePath)
				loaded, err := scene.Load(sp)
				if err != nil {
					return err
				}
				s = loaded
			}

			rt := runtime.NewFromScene(s)
			rt.StartPlay()

			w, err := rt.ActiveWorld()
			if err != nil {
				return err
			}

			frame := &render.Frame{
				ClearColor: color.RGBA{R: 15, G: 18, B: 24, A: 255},
				World:      w,
				ProjectDir: projectDir,
			}

			// WASM scripts (опционально): .kenga/scripts/game.wasm
			ctx := context.Background()
			sh := script.NewHost(func(format string, args ...any) {})
			wasmAbs := filepath.Join(projectDir, ".kenga", "scripts", "game.wasm")
			if _, err := os.Stat(wasmAbs); err == nil {
				_ = sh.LoadWASM(ctx, wasmAbs)
				sh.AttachWorld(w)
			}

			// v0: дефолтный бэкенд — простой desktop-рендер.
			var b render.Backend
			switch backend {
			case "", "ebiten":
				be := ebiten.New("GoEngineKenga Runtime", 1280, 720)
				be.OnUpdate = func(dt float64) {
					dtDur := time.Duration(dt * float64(time.Second))
					// v0: дергаем простую систему для демонстрации PlayWorld.
					delta := rt.Step()
					if aw, err := rt.ActiveWorld(); err == nil {
						runtime.SpinSystem(aw, delta)
						frame.World = aw
					}
					_ = sh.HotReloadIfChanged(ctx)
					_ = sh.Update(ctx, dtDur)
				}
				b = be
			case "webgpu":
				b = webgpu.New("GoEngineKenga Runtime (WebGPU)", 1280, 720)
			default:
				// webgpu будет подключён в отдельном todo (через build tags и реальную реализацию)
				b = ebiten.New("GoEngineKenga Runtime (fallback)", 1280, 720)
			}

			return b.RunLoop(frame)
		},
	}

	cmd.Flags().StringVar(&projectDir, "project", ".", "Project directory")
	cmd.Flags().StringVar(&scenePath, "scene", "", "Scene path (relative to project)")
	cmd.Flags().StringVar(&backend, "backend", "ebiten", "Render backend: ebiten|webgpu")

	return cmd
}

