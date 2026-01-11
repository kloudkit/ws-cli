package secrets

import (
	"github.com/kloudkit/ws-cli/cmd/secrets/generate"
	"github.com/spf13/cobra"
)

var SecretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage encryption, decryption, and vaults for secrets",
}

func init() {
	SecretsCmd.PersistentFlags().String("master", "", "Master key or path to key file")
	SecretsCmd.PersistentFlags().String("output", "", "Write output to file instead of stdout")
	SecretsCmd.PersistentFlags().String("mode", "", "File permissions (e.g., 0o600, 384), only when --output is used")
	SecretsCmd.PersistentFlags().Bool("force", false, "Overwrite existing files")
	SecretsCmd.PersistentFlags().Bool("raw", false, "Output without styling")

	SecretsCmd.AddCommand(encryptCmd, decryptCmd, generate.GenerateCmd, vaultCmd)
}
