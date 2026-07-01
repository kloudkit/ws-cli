package secrets

import "github.com/spf13/cobra"

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate master keys or login password hashes",
	Long:  "Generate the credentials the workspace needs — a master key for secrets, or a login password hash for the server.",
}

func init() {
	generateCmd.AddCommand(masterCmd, loginCmd)
}
