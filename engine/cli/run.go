package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"time"

	ebitenimg "github.com/hajimehoshi/ebiten/v2"
	"github.com/spf13/cobra"

	"goenginekenga/engine/api"
	"goenginekenga/engine/asset"
	"goenginekenga/engine/render"
	"goenginekenga/engine/render/ebiten"
	"goenginekenga/engine/render/headless"
	"goenginekenga/engine/render/webgpu"
	"goenginekenga/engine/runtime"
	"goenginekenga/engine/scene"
	"goenginekenga/engine/script"
)

func newRunCommand() *cobra.Command {
	var projectDir string
	var scenePath string
	var backend string
	var wsPort string
	var wsDisabled bool
	var headlessMode bool

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a project (runtime window)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Абсолютный путь проекта — для resolver и asset loading
			absProject, err := filepath.Abs(projectDir)
			if err != nil {
				absProject = projectDir
			}
			projectDir = absProject

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

			ctx := context.Background()
			sh := script.NewHost(func(format string, args ...any) {})
			wasmAbs := filepath.Join(projectDir, ".kenga", "scripts", "game.wasm")
			if _, err := os.Stat(wasmAbs); err == nil {
				_ = sh.LoadWASM(ctx, wasmAbs)
				sh.AttachWorld(w)
			}

			var resolver *asset.Resolver
			if r, err := asset.NewResolver(projectDir); err == nil {
				resolver = r
			}

			var watcher *asset.Watcher
			sceneRelPath := scenePath
			if sceneRelPath == "" {
				sceneRelPath = "scenes/main.scene.json"
			}
			if aw, err := asset.NewWatcher(projectDir, sceneRelPath); err == nil {
				watcher = aw
				defer watcher.Close()
			}

			var apiManager *api.Manager
			if !wsDisabled {
				apiManager = api.NewManager(rt, projectDir)

				addr := wsPort
				if addr == "" {
					addr = "127.0.0.1:7777"
				}
				if err := api.NewServerWithVersion(addr, apiManager, Version).Start(ctx); err != nil {
					return err
				}
			}

			scenePathAbs := filepath.Join(projectDir, sceneRelPath)
			var frame *render.Frame
			frame = &render.Frame{
				ClearColor: color.RGBA{R: 15, G: 18, B: 24, A: 255},
				World:      w,
				ProjectDir: projectDir,
				Resolver:   resolver,
				OnFrameRendered: func(screen interface{}) {
					if apiManager == nil || !apiManager.HasViewportSubscribers() {
						return
					}
					img, ok := screen.(*ebitenimg.Image)
					if !ok {
						return
					}
					b := img.Bounds()
					imgW, imgH := b.Dx(), b.Dy()
					pixels := make([]byte, 4*imgW*imgH)
					img.ReadPixels(pixels)
					rgba := &image.RGBA{Pix: pixels, Stride: imgW * 4, Rect: b}
					var buf bytes.Buffer
					if err := png.Encode(&buf, rgba); err != nil {
						return
					}
					apiManager.BroadcastViewportFrame(base64.StdEncoding.EncodeToString(buf.Bytes()))
				},
				OnUpdate: func(dt float64) {
					if apiManager != nil {
						apiManager.ProcessPending(ctx)
					}
					if watcher != nil {
						if watcher.ConsumeAssetsDirty() {
							if db, err := asset.Open(projectDir); err == nil {
								_, _ = db.ImportAll()
							}
						}
						if watcher.ConsumeIndexDirty() && resolver != nil {
							_ = resolver.Refresh()
						}
						if watcher.ConsumeSceneDirty() {
							if reloaded, err := scene.Load(scenePathAbs); err == nil {
								rt.ReplaceFromScene(reloaded)
								if world, err := rt.ActiveWorld(); err == nil {
									frame.World = world
								}
							}
						}
					}
					dtDur := time.Duration(dt * float64(time.Second))
					p := rt.GetProfiler()
					if p != nil {
						start := p.StartFrame()
						defer p.EndFrame(start)
						p.UpdateMemoryUsage()
					}
					delta := rt.Step()
					if aw, err := rt.ActiveWorld(); err == nil {
						runtime.SpinSystem(aw, delta)
						frame.World = aw
					}
					_ = sh.HotReloadIfChanged(ctx)
					_ = sh.Update(ctx, dtDur)
				},
			}

			// Headless: нет окна, только WebSocket API
			var b render.Backend
			if headlessMode {
				hb := headless.New(apiManager, rt, projectDir, sh)
				if watcher != nil {
					hb.SetWatcher(watcher)
					hb.SetScenePath(scenePathAbs)
				}
				b = hb
			} else {
				switch backend {
				case "", "ebiten":
					b = ebiten.New("GoEngineKenga Runtime", 1280, 720)
				case "webgpu":
					b = webgpu.New("GoEngineKenga Runtime (WebGPU)", 1280, 720)
				default:
					b = ebiten.New("GoEngineKenga Runtime (fallback)", 1280, 720)
				}
			}

			return b.RunLoop(frame)
		},
	}

	cmd.Flags().StringVar(&projectDir, "project", ".", "Project directory")
	cmd.Flags().StringVar(&scenePath, "scene", "", "Scene path (relative to project)")
	cmd.Flags().StringVar(&backend, "backend", "ebiten", "Render backend: ebiten|webgpu")
	cmd.Flags().StringVar(&wsPort, "ws-port", "127.0.0.1:7777", "WebSocket control listen address (empty to disable)")
	cmd.Flags().BoolVar(&wsDisabled, "no-ws", false, "Disable WebSocket control server")
	cmd.Flags().BoolVar(&headlessMode, "headless", false, "Run without window (WebSocket only, for KengaCAD)")

	return cmd
}
