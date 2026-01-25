package cli

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kenga",
		Short: "GoEngineKenga CLI",
	}

	cmd.AddCommand(
		newRunCommand(),
		newImportCommand(),
		newScriptCommand(),
		newNewCommand(),
	)

	return cmd
}
