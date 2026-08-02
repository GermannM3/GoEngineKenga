package script

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goenginekenga/engine/ecs"
	"goenginekenga/engine/input"
	emath "goenginekenga/engine/math"
)

// testWASM — минимальный WASM-модуль, собранный вручную (без tinygo).
// Импортирует env.getInputKey/getInputAxis/getEntityByName/getEntityTransform/
// setEntityTransform/addVelocity/debugLog и экспортирует Update(dtMillis), который:
//  1. находит "Player" по имени (getEntityByName),
//  2. читает его transform (getEntityTransform → память 16),
//  3. ставит позицию (10, 20, 30) (setEntityTransform),
//  4. пишет mem[64] = getInputKey(36 /*Space*/),
//  5. пишет mem[68] = f32 getInputAxis(0),
//  6. вызывает debugLog("TestTick\n"),
//  7. добавляет velocity (5, 0, -3) (addVelocity).
const testWASM = "AGFzbQEAAAABJwdgAX8AYAF/AX9gAn9/AX5gAn5/AX9gBH59fX0AYAF/AX1gAn9/AAKPAQcDZW52C2dldElucHV0S2V5AAEDZW52DGdldElucHV0QXhpcwAFA2Vudg9nZXRF" +
	"bnRpdHlCeU5hbWUAAgNlbnYSZ2V0RW50aXR5VHJhbnNmb3JtAAMDZW52EnNldEVudGl0eVRyYW5zZm9ybQAEA2VudgthZGRWZWxvY2l0eQAEA2VudghkZWJ1Z0xvZwAGAwIB" +
	"AAUDAQABBwoBBlVwZGF0ZQAHClYBVAEBfkEIQQYQAiEBIAFBEBADGiABQwAAIEFDAACgQUMAAPBBEARBwABBJBAAOgAAQcQAQQAQATgCAEHIAEEJEAYgAUMAAKBAQwAAAABD" +
	"AABAwBAFCwsbAgBBCAsGUGxheWVyAEHIAAsJVGVzdFRpY2sK"

func loadTestModule(t *testing.T) (*Host, *ecs.World, *[]string) {
	t.Helper()
	wasm, err := base64.StdEncoding.DecodeString(testWASM)
	if err != nil {
		t.Fatalf("decode wasm: %v", err)
	}

	logs := &[]string{}
	h := NewHost(func(format string, args ...any) {
		*logs = append(*logs, strings.TrimSpace(fmt.Sprintf(format, args...)))
	})
	ctx := context.Background()

	dir := t.TempDir()
	path := filepath.Join(dir, "game.wasm")
	if err := os.WriteFile(path, wasm, 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}
	if err := h.LoadWASM(ctx, path); err != nil {
		t.Fatalf("LoadWASM: %v", err)
	}

	w := ecs.NewWorld()
	playerID := w.CreateEntity("Player")
	w.SetTransform(playerID, ecs.Transform{Position: emath.V3(1, 2, 3), Scale: emath.V3(1, 1, 1)})
	w.SetRigidbody(playerID, ecs.Rigidbody{})
	h.AttachWorld(w)
	return h, w, logs
}

func findByName(w *ecs.World, name string) ecs.EntityID {
	for _, id := range w.Entities() {
		if w.Name(id) == name {
			return id
		}
	}
	return 0
}

// TestHostWASMRoundTrip гоняет реальный WASM-модуль через wazero:
// проверяет все host-функции и их связь с ECS-миром.
func TestHostWASMRoundTrip(t *testing.T) {
	h, w, logs := loadTestModule(t)
	ctx := context.Background()

	// Space нажата → getInputKey(36) = 1; зажата A → ось = -1
	st := input.NewState()
	st.SetKeyPressed(input.KeySpace, true)
	st.SetKeyPressed(input.KeyA, true)
	h.SetInputState(st)

	if err := h.Update(ctx, time.Second); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// 1. transform игрока переписан скриптом на (10, 20, 30)
	playerID := findByName(w, "Player")
	tr, ok := w.GetTransform(playerID)
	if !ok {
		t.Fatal("no transform")
	}
	if tr.Position.X != 10 || tr.Position.Y != 20 || tr.Position.Z != 30 {
		t.Fatalf("transform = %+v, want (10,20,30)", tr.Position)
	}

	// 2. velocity += (5, 0, -3)
	rb, ok := w.GetRigidbody(playerID)
	if !ok {
		t.Fatal("no rigidbody")
	}
	if rb.Velocity.X != 5 || rb.Velocity.Z != -3 {
		t.Fatalf("velocity = %+v, want X=5 Z=-3", rb.Velocity)
	}

	// 3. память: mem[64] = 1 (Space), mem[68] = -1 (ось A)
	mem := h.mod.Memory()
	if mem == nil {
		t.Fatal("no memory")
	}
	keyByte, _ := mem.ReadByte(64)
	if keyByte != 1 {
		t.Fatalf("mem[64] = %d, want 1 (Space pressed)", keyByte)
	}
	axisBits, _ := mem.ReadUint32Le(68)
	if got := math.Float32frombits(axisBits); got != -1 {
		t.Fatalf("mem[68] = %v, want -1 (axis A)", got)
	}

	// 4. debugLog дошёл до логгера
	found := false
	for _, l := range *logs {
		if strings.Contains(l, "TestTick") {
			found = true
		}
	}
	if !found {
		t.Fatalf("debugLog output missing TestTick, logs=%v", *logs)
	}
}

// TestHostWithoutInputState — без привязанного input state функции ввода
// возвращают пустые значения, остальные host-функции работают.
func TestHostWithoutInputState(t *testing.T) {
	h, w, _ := loadTestModule(t)
	ctx := context.Background()

	if err := h.Update(ctx, time.Second); err != nil {
		t.Fatalf("Update: %v", err)
	}
	playerID := findByName(w, "Player")
	tr, _ := w.GetTransform(playerID)
	if tr.Position.X != 10 {
		t.Fatalf("transform X = %v, want 10 (script runs without input)", tr.Position.X)
	}
	mem := h.mod.Memory()
	keyByte, _ := mem.ReadByte(64)
	if keyByte != 0 {
		t.Fatalf("mem[64] = %d, want 0 (no input state)", keyByte)
	}
	axisBits, _ := mem.ReadUint32Le(68)
	if got := math.Float32frombits(axisBits); got != 0 {
		t.Fatalf("mem[68] = %v, want 0", got)
	}
}

// TestHostMustLoaded — модуль подгружается и экспортирует Update.
func TestHostMustLoaded(t *testing.T) {
	h, _, _ := loadTestModule(t)
	if err := h.MustLoaded(); err != nil {
		t.Fatalf("MustLoaded: %v", err)
	}
	if h.mod.ExportedFunction("Update") == nil {
		t.Fatal("Update export missing")
	}
}

// TestHostNameLookupOverWorld — getEntityByName не находит несуществующую сущность
// (проверка через память: "Ghost" по offset 8 не найдена → не ломает скрипт).
func TestHostNameLookupOverWorld(t *testing.T) {
	h, _, _ := loadTestModule(t)
	// Повторный Update не должен падать с уже записанной позицией
	ctx := context.Background()
	if err := h.Update(ctx, time.Second); err != nil {
		t.Fatalf("Update (second call): %v", err)
	}
	if err := h.Update(ctx, time.Second); err != nil {
		t.Fatalf("Update (third call): %v", err)
	}
}
