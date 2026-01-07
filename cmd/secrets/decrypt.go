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
		force, _ := cmd.Flags().GetBool("force")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		raw, _ := cmd.Flags().GetBool("raw")

		if encrypted == "" && vaultPath == "" {
			return fmt.Errorf("either --encrypted or --vault is required")
		}

		key, err := internalSecrets.ResolveMasterKey(masterKey)
		if err != nil {
			return err
		}

		if encrypted != "" {
			return decryptSingleValue(cmd, encrypted, dest, key, force, dryRun, raw)
		}

		return decryptVault(cmd, vaultPath, key, force, dryRun)
	},
}

func decryptSingleValue(cmd *cobra.Command, encrypted, dest string, key []byte, force, dryRun, raw bool) error {
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

		if !raw {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Success().Render("Secret decrypted successfully"))
		}
		return nil
	}

	secret := &internalSecrets.Secret{
		Destination: dest,
	}

	opts := internalSecrets.WriteOptions{
		Force:  force,
		DryRun: dryRun,
	}

	if err := internalSecrets.WriteSecret(secret, decrypted, opts); err != nil {
		return err
	}

	if !dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n",
			styles.Success().Render(fmt.Sprintf("Secret written to %s", dest)))
	}

	return nil
}

func decryptVault(cmd *cobra.Command, vaultPath string, key []byte, force, dryRun bool) error {
	vault, err := internalSecrets.LoadVaultFromFile(vaultPath)
	if err != nil {
		return err
	}

	opts := internalSecrets.WriteOptions{
		Force:  force,
		DryRun: dryRun,
	}

	if err := vault.DecryptAll(key, opts); err != nil {
		return err
	}

	if !dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n",
			styles.Success().Render(fmt.Sprintf("Successfully decrypted %d secret(s) from vault", len(vault.Secrets))))
	}

	return nil
}

func init() {
	decryptCmd.Flags().String("encrypted", "", "Encrypted value to decrypt")
	decryptCmd.Flags().String("dest", "", "Destination (file, env, or stdout)")
	decryptCmd.Flags().String("vault", "", "Path to vault file")
	decryptCmd.Flags().Bool("raw", false, "Output without styling")

	decryptCmd.MarkFlagsMutuallyExclusive("encrypted", "vault")
}
