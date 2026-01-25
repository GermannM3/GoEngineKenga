package script

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type BuildOptions struct {
	ProjectDir string
	InFile     string // относительный путь внутри проекта, например scripts/game/main.go
	OutFile    string // относительный путь внутри проекта, например .kenga/scripts/game.wasm
}

// BuildTinyGoWASM собирает Go-скрипт в WASM через TinyGo.
// Требует установленный tinygo в PATH.
func BuildTinyGoWASM(opts BuildOptions) error {
	if opts.ProjectDir == "" {
		opts.ProjectDir = "."
	}
	if opts.InFile == "" {
		return fmt.Errorf("missing InFile")
	}
	if opts.OutFile == "" {
		return fmt.Errorf("missing OutFile")
	}

	tinygo, err := exec.LookPath("tinygo")
	if err != nil {
		return fmt.Errorf("tinygo not found in PATH (install TinyGo to build WASM)")
	}

	inAbs := filepath.Join(opts.ProjectDir, filepath.FromSlash(opts.InFile))
	outAbs := filepath.Join(opts.ProjectDir, filepath.FromSlash(opts.OutFile))

	if err := os.MkdirAll(filepath.Dir(outAbs), 0o755); err != nil {
		return err
	}

	// v0: target=wasi для максимальной переносимости (wazero)
	cmd := exec.Command(tinygo, "build", "-target=wasi", "-o", outAbs, inAbs)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
