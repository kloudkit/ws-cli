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
		secretType, _ := cmd.Flags().GetString("type")
		dest, _ := cmd.Flags().GetString("dest")
		vaultPath, _ := cmd.Flags().GetString("vault")
		masterKey, _ := cmd.Flags().GetString("master")
		force, _ := cmd.Flags().GetBool("force")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		verbose, _ := cmd.Flags().GetBool("verbose")

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

		if verbose {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Title().Render("Encrypt"))
			fmt.Fprintf(cmd.OutOrStdout(), "Value: %s, Type: %s, Dest: %s, Vault: %s, Force: %v, DryRun: %v\n", value, secretType, dest, vaultPath, force, dryRun)
		}

		// If no vault is specified, print to stdout
		if vaultPath == "" {
			fmt.Fprintln(cmd.OutOrStdout(), encrypted)
		} else {
			// TODO: Implement vault updating logic
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
}
