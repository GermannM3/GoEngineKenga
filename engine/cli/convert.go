package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"goenginekenga/engine/convert"
)

func newConvertCommand() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert IPT/IAM to glTF (requires Autodesk Forge and Node.js)",
		Long: `Convert Autodesk Inventor files (.ipt, .iam) to glTF/GLB.

Requires:
  - FORGE_CLIENT_ID and FORGE_CLIENT_SECRET (from aps.autodesk.com)
  - Node.js (for forge-convert-utils)

See docs/CAD_FORMATS.md for alternative conversion methods.
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]
			if output == "" {
				ext := filepath.Ext(input)
				output = input[:len(input)-len(ext)] + ".glb"
			}
			err := convert.ConvertIPTIAMToGLTF(input, output)
			if err != nil {
				return err
			}
			fmt.Println("Saved:", output)
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output path (.glb or .gltf)")
	return cmd
}
