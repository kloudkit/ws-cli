package secrets

import (
	"fmt"

	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/spf13/cobra"
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Encrypt a secret",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := newContext(cmd)
		value := getString(cmd, "value")
		vaultPath := getString(cmd, "vault")
		dest := getString(cmd, "dest")
		secretType := getString(cmd, "type")

		if value == "" {
			return fmt.Errorf("value is required")
		}

		masterKey, err := ctx.resolveMasterKey()
		if err != nil {
			return err
		}

		if vaultPath == "" {
			encrypted, err := internalSecrets.Encrypt([]byte(value), masterKey)
			if err != nil {
				return fmt.Errorf("encryption failed: %w", err)
			}
			ctx.print(encrypted)
			return nil
		}

		if dest == "" {
			return fmt.Errorf("--dest is required when using --vault")
		}

		if err := internalSecrets.EncryptToVault([]byte(value), vaultPath, dest, secretType, masterKey, ctx.force, ctx.dryRun); err != nil {
			return err
		}

		if ctx.dryRun {
			ctx.dryRunMsg("[DRY-RUN] Would add secret to vault " + vaultPath)
		} else {
			ctx.success(fmt.Sprintf("Secret added to vault %s", vaultPath))
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
