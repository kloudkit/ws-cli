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
	SecretsCmd.PersistentFlags().String("mode", "", "File permissions (e.g., 0o600, 384) - only when --output is used")
	SecretsCmd.PersistentFlags().Bool("force", false, "Overwrite existing files/values")
	SecretsCmd.PersistentFlags().Bool("raw", false, "Output without styling")

	SecretsCmd.AddCommand(encryptCmd, decryptCmd, generateCmd)
}
