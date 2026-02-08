package headless

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"goenginekenga/engine/api"
	"goenginekenga/engine/asset"
	"goenginekenga/engine/render"
	"goenginekenga/engine/runtime"
	"goenginekenga/engine/scene"
	"goenginekenga/engine/script"
)

// Backend runs the engine without opening a window (WebSocket only).
// Used when KengaCAD controls the UI and engine runs in background.
type Backend struct {
	frame      *render.Frame
	apiManager *api.Manager
	rt         *runtime.Runtime
	projectDir string
	scenePath  string
	sh         *script.Host
	watcher   *asset.Watcher
	tickRate   time.Duration
}

// New creates a headless backend.
func New(apiManager *api.Manager, rt *runtime.Runtime, projectDir string, sh *script.Host) *Backend {
	return &Backend{
		apiManager: apiManager,
		rt:         rt,
		projectDir: projectDir,
		scenePath:  filepath.Join(projectDir, "scenes", "main.scene.json"),
		sh:         sh,
		tickRate:   time.Second / 60,
	}
}

// SetWatcher устанавливает watcher для hot-reload ассетов и сцен.
func (b *Backend) SetWatcher(w *asset.Watcher) { b.watcher = w }

// SetScenePath задаёт путь к сцене для hot-reload.
func (b *Backend) SetScenePath(p string) { b.scenePath = p }

// RunLoop runs the engine loop without display. Processes WebSocket commands
// and steps the runtime. Exits on SIGINT or when parent process exits.
func (b *Backend) RunLoop(initial *render.Frame) error {
	b.frame = initial
	ctx := context.Background()
	ticker := time.NewTicker(b.tickRate)
	defer ticker.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sigCh:
			return nil
		case <-ticker.C:
			if b.apiManager != nil {
				b.apiManager.ProcessPending(ctx)
			}
			if b.watcher != nil && b.frame != nil {
				if b.watcher.ConsumeAssetsDirty() {
					if db, err := asset.Open(b.projectDir); err == nil {
						_, _ = db.ImportAll()
					}
				}
				if b.watcher.ConsumeIndexDirty() && b.frame.Resolver != nil {
					if r, ok := b.frame.Resolver.(interface{ Refresh() error }); ok {
						_ = r.Refresh()
					}
				}
				if b.watcher.ConsumeSceneDirty() {
					if reloaded, err := scene.Load(b.scenePath); err == nil {
						b.rt.ReplaceFromScene(reloaded)
						if aw, err := b.rt.ActiveWorld(); err == nil {
							b.frame.World = aw
						}
					}
				}
			}
			delta := b.rt.Step()
			if aw, err := b.rt.ActiveWorld(); err == nil {
				runtime.SpinSystem(aw, delta)
				if b.frame != nil {
					b.frame.World = aw
				}
			}
			if b.sh != nil {
				_ = b.sh.HotReloadIfChanged(ctx)
				_ = b.sh.Update(ctx, b.tickRate)
			}
		}
	}
}
