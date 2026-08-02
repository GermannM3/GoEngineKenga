package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"time"

	ebitenimg "github.com/hajimehoshi/ebiten/v2"
	"github.com/spf13/cobra"

	"goenginekenga/engine/animation"
	"goenginekenga/engine/api"
	"goenginekenga/engine/asset"
	"goenginekenga/engine/audio"
	"goenginekenga/engine/ecs"
	"goenginekenga/engine/gameplay"
	"goenginekenga/engine/input"
	"goenginekenga/engine/project"
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
			// Нормализация пути: в Git Bash D:\CyberNinja приходит как D:CyberNinja (слэш съедается),
			// и Go на Windows трактует это как относительный путь к текущей папке на диске D:
			projectDir = normalizeProjectPath(projectDir)
			absProject, err := filepath.Abs(projectDir)
			if err != nil {
				absProject = projectDir
			}
			projectDir = absProject

			var s *scene.Scene
			if scenePath == "" {
				// Первая сцена из project.kenga.json или дефолт
				if p, err := project.Load(projectDir); err == nil && len(p.Scenes) > 0 {
					scenePath = p.Scenes[0]
				} else {
					scenePath = "scenes/main.scene.json"
				}
			}
			if scenePath != "" {
				sp := filepath.Join(projectDir, scenePath)
				loaded, err := scene.Load(sp)
				if err != nil {
					fmt.Fprintf(os.Stderr, "kenga: load scene %s: %v (using default scene)\n", sp, err)
					s = scene.DefaultScene()
				} else {
					s = loaded
					if s != nil && len(s.Entities) > 0 {
						fmt.Fprintf(os.Stderr, "kenga: scene %q from %s (%d entities)\n", s.Name, sp, len(s.Entities))
					}
				}
			} else {
				s = scene.DefaultScene()
			}
			if s == nil {
				s = scene.DefaultScene()
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
			// Create game logic system
			gameLogicSystem := gameplay.NewGameLogicSystem()

			// Аудио: AudioSource из ECS → движок звука (клипы по asset ID через резолвер)
			audioSystem := audio.NewAudioSystem(resolver)
			if gameLogicSystem != nil {
				gameLogicSystem.Sound = audioSystem
			}

			// Create animation system
			animationSystem := animation.NewAnimationSystem()
			// 3D skeletal-анимации (glTF skins): пишет bone matrices в ecs.Animator
			skeletalSystem := animation.NewSkeletalAnimationSystem(projectDir)

			clearColor := color.RGBA{R: 15, G: 18, B: 24, A: 255}
			if gameplay.HasKart(w) {
				clearColor = color.RGBA{R: 28, G: 78, B: 38, A: 255} // трава
			}

			// Единый шаг игровых систем (ввод → анимация → геймплей).
			// Используется и оконным OnUpdate, и headless-бэкендом для паритета поведения.
			systemsUpdate := func(aw *ecs.World, is *input.State, dt float64) {
				if gameplay.HasKart(aw) {
					gameplay.KartSystem(aw, is, float32(dt))
				} else {
					runtime.ApplyPlayerInput(aw, is, float32(dt))
				}
				animationSystem.Update(aw)
				skeletalSystem.Update(aw, float32(dt))
				if !gameplay.HasKart(aw) {
					gameLogicSystem.Update(aw, is)
				}
				audioSystem.Update(aw, time.Duration(dt*float64(time.Second)))
			}

			var frame *render.Frame
			frame = &render.Frame{
				ClearColor: clearColor,
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
							frame.InvalidateMeshCache = true
						}
						if watcher.ConsumeIndexDirty() && resolver != nil {
							_ = resolver.Refresh()
							frame.InvalidateMeshCache = true
						}
						if watcher.ConsumeSceneDirty() {
							if reloaded, err := scene.Load(scenePathAbs); err == nil {
								rt.ReplaceFromScene(reloaded)
								if world, err := rt.ActiveWorld(); err == nil {
									frame.World = world
								}
								frame.OrbitResetRequested = true
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
					if aw, err := rt.ActiveWorld(); err == nil {
						if inputState, ok := frame.InputState.(*input.State); ok {
							sh.SetInputState(inputState)
							systemsUpdate(aw, inputState, dt)
						} else {
							systemsUpdate(aw, &input.State{}, dt)
						}
						sh.AttachWorld(aw) // мир мог пересоздаться при hot-reload сцены
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
				hb.SetOnTick(func(dt float64) {
					if aw, err := rt.ActiveWorld(); err == nil {
						systemsUpdate(aw, &input.State{}, dt)
					}
				})
				if watcher != nil {
					hb.SetWatcher(watcher)
					hb.SetScenePath(scenePathAbs)
				}
				b = hb
			} else {
				switch backend {
				case "", "ebiten":
					b = ebiten.New("GoEngineKenga Runtime", 1280, 720)

					// Проверяем, есть ли в сцене спрайты, и включаем 2D рендеринг если есть
					hasSprites := false
					for _, id := range w.Entities() {
						if _, ok := w.GetSpriteRenderer(id); ok {
							hasSprites = true
							break
						}
					}
					if hasSprites {
						if ebitenBackend, ok := b.(*ebiten.Backend); ok {
							ebitenBackend.Enable2DSprites(true)
							ebitenBackend.Enable3D(false)
							ebitenBackend.EnableOrbitCamera(false)
							if gameplay.HasKart(w) {
								ebitenBackend.SetSpriteRenderer(ebiten.NewSpriteRenderSystem(projectDir))
							}
						}
					}
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

// normalizeProjectPath исправляет путь на Windows, когда из Git Bash приходит D:CyberNinja
// (слэш после двоеточия съедается) — иначе filepath.Abs трактует это как относительный путь.
func normalizeProjectPath(path string) string {
	if path == "" || path == "." {
		return path
	}
	// Явная проверка: буква диска + ":" + путь без слэша в начале (D:CyberNinja, D:foo/bar)
	if len(path) >= 2 && path[1] == ':' {
		letter := path[0]
		if (letter >= 'A' && letter <= 'Z') || (letter >= 'a' && letter <= 'z') {
			rest := path[2:]
			if len(rest) > 0 && rest[0] != '/' && rest[0] != filepath.Separator {
				return string(letter) + ":" + string(filepath.Separator) + rest
			}
		}
	}
	vol := filepath.VolumeName(path)
	if vol == "" || len(path) <= len(vol) {
		return path
	}
	rest := path[len(vol):]
	if rest == "" {
		return path
	}
	if rest[0] == filepath.Separator || rest[0] == '/' {
		return path
	}
	return vol + string(filepath.Separator) + rest
}
