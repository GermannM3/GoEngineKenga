package script

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"goenginekenga/engine/ecs"
)

type Host struct {
	mu sync.Mutex

	rt  wazero.Runtime
	mod api.Module

	wasmPath string
	lastStat time.Time

	logf func(format string, args ...any)

	world *ecs.World
}

func NewHost(logf func(string, ...any)) *Host {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Host{logf: logf}
}

func (h *Host) AttachWorld(w *ecs.World) { h.world = w }

func (h *Host) LoadWASM(ctx context.Context, wasmPath string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rt == nil {
		h.rt = wazero.NewRuntime(ctx)
		if err := h.registerHostFunctions(ctx); err != nil {
			return err
		}
	}

	if h.mod != nil {
		_ = h.mod.Close(ctx)
		h.mod = nil
	}

	wasm, err := os.ReadFile(wasmPath)
	if err != nil {
		return err
	}
	mod, err := h.rt.Instantiate(ctx, wasm)
	if err != nil {
		return err
	}
	h.mod = mod
	h.wasmPath = wasmPath

	if st, err := os.Stat(wasmPath); err == nil {
		h.lastStat = st.ModTime()
	}
	h.logf("script: loaded %s\n", wasmPath)
	return nil
}

func (h *Host) registerHostFunctions(ctx context.Context) error {
	_, err := h.rt.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithFunc(h.hostDebugLog).
		Export("debugLog").
		Instantiate(ctx)
	return err
}

// debugLog(ptr, len) — пишет строку в консоль редактора/рантайма.
func (h *Host) hostDebugLog(ctx context.Context, m api.Module, ptr, l uint32) {
	mem := m.Memory()
	if mem == nil {
		return
	}
	b, ok := mem.Read(ptr, l)
	if !ok {
		return
	}
	h.logf("%s", string(b))
}

// Update вызывает экспортированную функцию `Update` (если есть).
// v0: Update(dtMillis int32)
func (h *Host) Update(ctx context.Context, dt time.Duration) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.mod == nil {
		return nil
	}
	fn := h.mod.ExportedFunction("Update")
	if fn == nil {
		return nil
	}
	_, err := fn.Call(ctx, uint64(dt.Milliseconds()))
	return err
}

// HotReloadIfChanged перезагружает wasm при изменении файла.
func (h *Host) HotReloadIfChanged(ctx context.Context) error {
	h.mu.Lock()
	wasmPath := h.wasmPath
	last := h.lastStat
	h.mu.Unlock()

	if wasmPath == "" {
		return nil
	}
	st, err := os.Stat(wasmPath)
	if err != nil {
		return nil
	}
	if st.ModTime().After(last) {
		return h.LoadWASM(ctx, wasmPath)
	}
	return nil
}

func (h *Host) Close(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.mod != nil {
		_ = h.mod.Close(ctx)
		h.mod = nil
	}
	if h.rt != nil {
		err := h.rt.Close(ctx)
		h.rt = nil
		return err
	}
	return nil
}

func (h *Host) MustLoaded() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.mod == nil {
		return fmt.Errorf("no wasm loaded")
	}
	return nil
}
