package secrets

import (
	"fmt"

	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Encrypt a secret",
	RunE: func(cmd *cobra.Command, args []string) error {
		value, _ := cmd.Flags().GetString("value")
		vaultPath, _ := cmd.Flags().GetString("vault")
		masterKey, _ := cmd.Flags().GetString("master")
		raw, _ := cmd.Flags().GetBool("raw")

		if value == "" {
			return fmt.Errorf("value is required")
		}

		key, err := internalSecrets.ResolveMasterKey(masterKey)
		if err != nil {
			return err
		}

		encrypted, err := internalSecrets.Encrypt([]byte(value), key)
		if err != nil {
			return fmt.Errorf("encryption failed: %w", err)
		}

		if vaultPath == "" {
			if raw {
				fmt.Fprintln(cmd.OutOrStdout(), encrypted)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), styles.Code().Render(encrypted))
			}
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "Vault updating logic not yet implemented")
		}

		return nil
	},
}

func init() {
	encryptCmd.Flags().String("value", "", "Value to encrypt")
	encryptCmd.Flags().String("type", "", "Type of secret (kubeconfig, ssh, env, etc.)")
	encryptCmd.Flags().String("dest", "", "Destination file or environment variable")
	encryptCmd.Flags().String("vault", "", "Path to vault file")
	encryptCmd.Flags().Bool("raw", false, "Output without styling")
}
