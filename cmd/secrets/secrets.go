package secrets

import (
	"github.com/spf13/cobra"
)

var SecretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage encryption, decryption, and vaults for secrets",
}

func init() {
	SecretsCmd.PersistentFlags().String("master", "", "Master key or path to key file")
	SecretsCmd.PersistentFlags().Bool("force", false, "Overwrite existing files/values")
	SecretsCmd.PersistentFlags().Bool("dry-run", false, "Perform operation without writing changes")

	SecretsCmd.AddCommand(encryptCmd, decryptCmd, generateCmd, vaultCmd)
}
