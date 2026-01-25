package main

import (
	"os"

	"goenginekenga/engine/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
