package cli

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"goenginekenga/engine/script"
)

func newScriptCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "script",
		Short: "Script tooling (build, etc.)",
	}

	cmd.AddCommand(newScriptBuildCommand())
	return cmd
}

func newScriptBuildCommand() *cobra.Command {
	var projectDir string
	var inFile string
	var outFile string

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build scripts (TinyGo -> WASM) for hot-reload",
		RunE: func(cmd *cobra.Command, args []string) error {
			if inFile == "" {
				inFile = filepath.ToSlash("scripts/game/main.go")
			}
			if outFile == "" {
				outFile = filepath.ToSlash(".kenga/scripts/game.wasm")
			}
			return script.BuildTinyGoWASM(script.BuildOptions{
				ProjectDir: projectDir,
				InFile:     inFile,
				OutFile:    outFile,
			})
		},
	}

	cmd.Flags().StringVar(&projectDir, "project", ".", "Project directory")
	cmd.Flags().StringVar(&inFile, "in", "", "Input Go file (relative to project)")
	cmd.Flags().StringVar(&outFile, "out", "", "Output WASM file (relative to project)")
	return cmd
}
