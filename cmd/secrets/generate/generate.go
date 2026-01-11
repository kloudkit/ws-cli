package generate

import "github.com/spf13/cobra"

var GenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate master keys or login password hashes",
}

func init() {
	GenerateCmd.AddCommand(masterCmd, loginCmd)
}
