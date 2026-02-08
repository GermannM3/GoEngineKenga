package main

import (
	"os"

	"goenginekenga/engine/cli"
)

var version = "dev"

func main() {
	cli.Version = version
	root := cli.NewRootCommand()
	root.Version = version
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
