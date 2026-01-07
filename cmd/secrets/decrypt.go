package secrets

import (
	"fmt"

	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypt secrets",
	RunE: func(cmd *cobra.Command, args []string) error {
		encrypted, _ := cmd.Flags().GetString("encrypted")
		dest, _ := cmd.Flags().GetString("dest")
		vaultPath, _ := cmd.Flags().GetString("vault")
		masterKey, _ := cmd.Flags().GetString("master")
		raw, _ := cmd.Flags().GetBool("raw")

		if encrypted == "" && vaultPath == "" {
			return fmt.Errorf("either --encrypted or --vault is required")
		}

		if encrypted != "" {
			key, err := internalSecrets.ResolveMasterKey(masterKey)
			if err != nil {
				return err
			}

			decrypted, err := internalSecrets.Decrypt(encrypted, key)
			if err != nil {
				return fmt.Errorf("decryption failed: %w", err)
			}

			if dest == "" || dest == "stdout" {
				if raw {
					fmt.Fprintln(cmd.OutOrStdout(), string(decrypted))
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), styles.Code().Render(string(decrypted)))
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "File/env writing logic not yet implemented")
			}

			if !raw && (dest == "" || dest == "stdout") {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Success().Render("Secret decrypted successfully"))
			}
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "Vault decryption logic not yet implemented")
		}

		return nil
	},
}

func init() {
	decryptCmd.Flags().String("encrypted", "", "Encrypted value to decrypt")
	decryptCmd.Flags().String("dest", "", "Destination (file, env, or stdout)")
	decryptCmd.Flags().String("vault", "", "Path to vault file")
	decryptCmd.Flags().Bool("raw", false, "Output without styling")

	decryptCmd.MarkFlagsMutuallyExclusive("encrypted", "vault")
}
