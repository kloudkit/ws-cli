package secrets

import "github.com/spf13/cobra"

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate master keys or login password hashes",
}

func init() {
	generateCmd.AddCommand(masterCmd, loginCmd)
}
