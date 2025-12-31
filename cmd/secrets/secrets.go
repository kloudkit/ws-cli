package secrets

import (
	"github.com/spf13/cobra"
)

var SecretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage encryption, decryption, and vaults for secrets",
}

func init() {
	SecretsCmd.AddCommand(encryptCmd, decryptCmd, generateCmd, vaultCmd)
}
