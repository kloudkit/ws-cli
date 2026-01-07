package secrets

import (
	"fmt"

	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/spf13/cobra"
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypt secrets",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := newContext(cmd)
		encrypted := getString(cmd, "encrypted")
		dest := getString(cmd, "dest")
		vaultPath := getString(cmd, "vault")

		if encrypted == "" && vaultPath == "" {
			return fmt.Errorf("either --encrypted or --vault is required")
		}

		masterKey, err := ctx.resolveMasterKey()
		if err != nil {
			return err
		}

		if encrypted != "" {
			decrypted, err := internalSecrets.DecryptSingle(encrypted, dest, masterKey, ctx.force, ctx.dryRun)
			if err != nil {
				return err
			}

			if dest == "" || dest == "stdout" {
				ctx.print(string(decrypted))
				if !ctx.raw {
					ctx.success("Secret decrypted successfully")
				}
			} else if !ctx.dryRun {
				ctx.success(fmt.Sprintf("Secret written to %s", dest))
			}

			return nil
		}

		if err := internalSecrets.DecryptVault(vaultPath, masterKey, ctx.force, ctx.dryRun); err != nil {
			return err
		}

		if !ctx.dryRun {
			vault, _ := internalSecrets.LoadVaultFromFile(vaultPath)
			ctx.success(fmt.Sprintf("Successfully decrypted %d secret(s) from vault", len(vault.Secrets)))
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
