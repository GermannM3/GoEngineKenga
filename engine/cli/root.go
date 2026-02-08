package cli

import "github.com/spf13/cobra"

// Version — версия движка (подставляется при сборке через -ldflags).
var Version = "dev"

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kenga",
		Short: "GoEngineKenga CLI",
	}

	cmd.AddCommand(
		newRunCommand(),
		newImportCommand(),
		newConvertCommand(),
		newScriptCommand(),
		newNewCommand(),
		newEditorCommand(),
	)

	return cmd
}
