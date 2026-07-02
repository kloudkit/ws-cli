package secrets

import (
	"github.com/spf13/cobra"
)

var SecretsCmd = &cobra.Command{
	Use:         "secrets",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Manage encryption and decryption of secrets",
	Long:        "Encrypt and decrypt values under a master key, and generate the keys themselves. Encrypted values are what the seed engine's secrets: map stores and decrypts at boot.",
	Example: `# Generate a master key
ws secrets generate master

# Encrypt a value under it
ws secrets encrypt "s3cr3t" --master ~/.ws/master.key`,
}

func init() {
	SecretsCmd.PersistentFlags().String("master", "", "Master key or path to key file")
	SecretsCmd.PersistentFlags().String("output", "", "Write output to file instead of stdout")
	SecretsCmd.PersistentFlags().String("mode", "", "File permissions (e.g., 0o600, 384), only when --output is used")
	SecretsCmd.PersistentFlags().Bool("force", false, "Overwrite existing files")
	SecretsCmd.PersistentFlags().Bool("raw", false, "Output without styling")

	SecretsCmd.AddCommand(encryptCmd, decryptCmd, generateCmd)
}
