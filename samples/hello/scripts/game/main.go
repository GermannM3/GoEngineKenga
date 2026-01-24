package main

import "unsafe"

//go:wasmimport env debugLog
func debugLog(ptr uint32, l uint32)

//export Update
func Update(dtMillis int32) {
	_ = dtMillis
	msg := []byte("Hello from WASM script!\\n")
	debugLog(uint32(uintptr(unsafe.Pointer(&msg[0]))), uint32(len(msg)))
}

func main() {}

