package vault

import "github.com/spf13/cobra"

var VaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Manage vault secrets",
}

func init() {
	VaultCmd.PersistentFlags().String("input", "", "Path to vault file")

	VaultCmd.AddCommand(lsCmd, decryptCmd, rotateCmd)
}
