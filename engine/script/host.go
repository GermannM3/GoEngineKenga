package script

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"goenginekenga/engine/ecs"
	"goenginekenga/engine/input"
	emath "goenginekenga/engine/math"
)

type Host struct {
	mu sync.Mutex

	rt  wazero.Runtime
	mod api.Module

	wasmPath string
	lastStat time.Time

	logf func(format string, args ...any)

	world      *ecs.World
	inputState *input.State
}

func NewHost(logf func(string, ...any)) *Host {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Host{logf: logf}
}

func (h *Host) AttachWorld(w *ecs.World) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.world = w
}

// SetInputState привязывает состояние ввода текущего кадра
// (доступно скрипту через getInputKey/getInputAxis).
func (h *Host) SetInputState(is *input.State) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.inputState = is
}

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
		NewFunctionBuilder().
		WithFunc(h.hostGetInputKey).
		Export("getInputKey").
		NewFunctionBuilder().
		WithFunc(h.hostGetInputAxis).
		Export("getInputAxis").
		NewFunctionBuilder().
		WithFunc(h.hostGetEntityByName).
		Export("getEntityByName").
		NewFunctionBuilder().
		WithFunc(h.hostGetEntityTransform).
		Export("getEntityTransform").
		NewFunctionBuilder().
		WithFunc(h.hostSetEntityTransform).
		Export("setEntityTransform").
		NewFunctionBuilder().
		WithFunc(h.hostAddVelocity).
		Export("addVelocity").
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

// getInputKey(key int32) i32 — нажата ли клавиша (коды из engine/input: 0..KeyMax).
// Возвращает 0/1: wazero не поддерживает bool как wasm-тип.
func (h *Host) hostGetInputKey(ctx context.Context, m api.Module, key int32) uint32 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.inputState == nil || key < 0 {
		return 0
	}
	if h.inputState.IsKeyPressed(input.Key(key)) {
		return 1
	}
	return 0
}

// getInputAxis(axis int32) f32 — ось ввода: 0 = горизонталь (A/D/←/→), 1 = вертикаль (W/S/↑/↓).
func (h *Host) hostGetInputAxis(ctx context.Context, m api.Module, axis int32) float32 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.inputState == nil {
		return 0
	}
	st := h.inputState
	switch axis {
	case 0:
		var v float32
		if st.IsKeyPressed(input.KeyA) || st.IsKeyPressed(input.KeyArrowLeft) {
			v -= 1
		}
		if st.IsKeyPressed(input.KeyD) || st.IsKeyPressed(input.KeyArrowRight) {
			v += 1
		}
		return v
	case 1:
		var v float32
		if st.IsKeyPressed(input.KeyS) || st.IsKeyPressed(input.KeyArrowDown) {
			v -= 1
		}
		if st.IsKeyPressed(input.KeyW) || st.IsKeyPressed(input.KeyArrowUp) {
			v += 1
		}
		return v
	default:
		return 0
	}
}

// getEntityByName(ptr, len) i64 — ID сущности по имени (0 = не найдена).
func (h *Host) hostGetEntityByName(ctx context.Context, m api.Module, ptr, l uint32) uint64 {
	mem := m.Memory()
	if mem == nil {
		return 0
	}
	b, ok := mem.Read(ptr, l)
	if !ok {
		return 0
	}
	name := string(b)

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.world == nil {
		return 0
	}
	for _, id := range h.world.Entities() {
		if h.world.Name(id) == name {
			return uint64(id)
		}
	}
	return 0
}

// getEntityTransform(id i64, outPtr u32) i32 — пишет position (3×f32 LE) по outPtr,
// возвращает 0/1 (успех).
func (h *Host) hostGetEntityTransform(ctx context.Context, m api.Module, id uint64, outPtr uint32) uint32 {
	mem := m.Memory()
	if mem == nil {
		return 0
	}

	h.mu.Lock()
	if h.world == nil {
		h.mu.Unlock()
		return 0
	}
	tr, ok := h.world.GetTransform(ecs.EntityID(id))
	h.mu.Unlock()
	if !ok {
		return 0
	}

	buf := make([]byte, 12)
	binary.LittleEndian.PutUint32(buf[0:], math.Float32bits(tr.Position.X))
	binary.LittleEndian.PutUint32(buf[4:], math.Float32bits(tr.Position.Y))
	binary.LittleEndian.PutUint32(buf[8:], math.Float32bits(tr.Position.Z))
	if !mem.Write(outPtr, buf) {
		return 0
	}
	return 1
}

// setEntityTransform(id i64, x f32, y f32, z f32) — задаёт позицию сущности.
func (h *Host) hostSetEntityTransform(ctx context.Context, m api.Module, id uint64, x, y, z float32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.world == nil {
		return
	}
	tr, ok := h.world.GetTransform(ecs.EntityID(id))
	if !ok {
		tr = ecs.Transform{Scale: emath.V3(1, 1, 1)}
	}
	tr.Position = emath.V3(x, y, z)
	h.world.SetTransform(ecs.EntityID(id), tr)
}

// addVelocity(id i64, x, y, z f32) — добавляет импульс к rigidbody сущности.
func (h *Host) hostAddVelocity(ctx context.Context, m api.Module, id uint64, x, y, z float32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.world == nil {
		return
	}
	rb, ok := h.world.GetRigidbody(ecs.EntityID(id))
	if !ok {
		return
	}
	rb.Velocity.X += x
	rb.Velocity.Y += y
	rb.Velocity.Z += z
	h.world.SetRigidbody(ecs.EntityID(id), rb)
}

// Update вызывает экспортированную функцию `Update` (если есть).
// v0: Update(dtMillis int32)
func (h *Host) Update(ctx context.Context, dt time.Duration) error {
	h.mu.Lock()
	if h.mod == nil {
		h.mu.Unlock()
		return nil
	}
	fn := h.mod.ExportedFunction("Update")
	h.mu.Unlock()

	if fn == nil {
		return nil
	}
	// Мьютекс на время вызова НЕ удерживается: wasm-код синхронно вызывает
	// host-функции (getInputKey, getEntityByName, ...), которые берут h.mu —
	// повторный Lock из той же горутины был бы взаимной блокировкой.
	// Цикл игры вызывает Update и HotReloadIfChanged последовательно,
	// поэтому подмена h.mod между Unlock и fn.Call не грозит.
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
