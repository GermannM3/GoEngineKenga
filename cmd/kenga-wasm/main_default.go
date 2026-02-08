//go:build !js

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "kenga-wasm: build with GOOS=js GOARCH=wasm for WebAssembly")
	os.Exit(1)
}
