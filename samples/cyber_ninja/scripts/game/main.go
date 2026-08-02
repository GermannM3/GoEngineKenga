//go:build wasm

// WASM-скрипт CyberNinja — «код игры на WASM» через host-функции движка.
// Сборка: kenga script build --project samples/cyber_ninja (нужен TinyGo).
//
// Демонстрирует:
//   - getEntityByName/getEntityTransform/setEntityTransform — патруль врага ScriptBot;
//   - getInputKey — прыжок игрока по Space (импульс через addVelocity);
//   - debugLog — периодический лог в консоль.
package main

import "unsafe"

//go:wasmimport env debugLog
func debugLog(ptr uint32, l uint32)

//go:wasmimport env getInputKey
func getInputKey(key int32) bool

//go:wasmimport env getInputAxis
func getInputAxis(axis int32) float32

//go:wasmimport env getEntityByName
func getEntityByName(ptr uint32, l uint32) uint64

//go:wasmimport env getEntityTransform
func getEntityTransform(id uint64, outPtr uint32) bool

//go:wasmimport env setEntityTransform
func setEntityTransform(id uint64, x float32, y float32, z float32)

//go:wasmimport env addVelocity
func addVelocity(id uint64, x float32, y float32, z float32)

// Коды клавиш совпадают с engine/input (KeySpace = 36, KeyA = 0, KeyD = 3).
const (
	keySpace     = 36
	axisH        = 0
	axisV        = 1
	botMinX      = float32(-7.0)
	botMaxX      = float32(7.0)
	botSpeed     = float32(3.0)
	jumpVelocity = float32(300.0)
)

var posBuf [12]byte // x, y, z как float32 (little-endian)

var (
	spacePrev bool
	tickCount int32
)

func cstr(s string) (uint32, uint32) {
	b := []byte(s)
	return uint32(uintptr(unsafe.Pointer(&b[0]))), uint32(len(b))
}

func f32(buf []byte, off int) float32 {
	return *(*float32)(unsafe.Pointer(&buf[off]))
}

//export Update
func Update(dtMillis int32) {
	tickCount++
	dt := float32(dtMillis) / 1000.0

	// Патруль ScriptBot: движение по X между botMinX и botMaxX.
	bot := getEntityByName(cstr("ScriptBot"))
	if bot != 0 {
		base := uint32(uintptr(unsafe.Pointer(&posBuf[0])))
		if getEntityTransform(bot, base) {
			x := f32(posBuf[:], 0)
			y := f32(posBuf[:], 4)
			z := f32(posBuf[:], 8)
			x += dt * botSpeed
			if x > botMaxX {
				x = botMinX
			}
			setEntityTransform(bot, x, y, z)
		}
	}

	// Прыжок игрока по Space (по фронту нажатия).
	player := getEntityByName(cstr("Player"))
	if player != 0 {
		space := getInputKey(keySpace)
		if space && !spacePrev {
			addVelocity(player, 0, jumpVelocity, 0)
		}
		spacePrev = space
	}

	// Лог раз в секунду (~60 тиков).
	if tickCount%60 == 0 {
		h := getInputAxis(axisH)
		v := getInputAxis(axisV)
		msg := []byte("CyberNinja — WASM tick, axis=(" + itoa(int(h)) + "," + itoa(int(v)) + ")\n")
		debugLog(uint32(uintptr(unsafe.Pointer(&msg[0]))), uint32(len(msg)))
	}
}

// itoa — минимальная конверсия int → string (для логов; tinygo без strconv-хелперов).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func main() {}
