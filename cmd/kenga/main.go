package main

import (
	"os"

	"goenginekenga/engine/cli"

	// Демо-скрипты сэмплов (spinner/patrol_x/bob). В своей игре импортируйте
	// пакет со своими скриптами в main своего exe.
	_ "goenginekenga/samples/cyber_ninja/scripts"
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
