package secrets

import (
	"fmt"
	"os"

	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Encrypt a secret",
	RunE: func(cmd *cobra.Command, args []string) error {
		value, _ := cmd.Flags().GetString("value")
		vaultPath, _ := cmd.Flags().GetString("vault")
		secretType, _ := cmd.Flags().GetString("type")
		dest, _ := cmd.Flags().GetString("dest")
		masterKey, _ := cmd.Flags().GetString("master")
		force, _ := cmd.Flags().GetBool("force")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
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
			return nil
		}

		return addToVault(cmd, vaultPath, encrypted, secretType, dest, force, dryRun)
	},
}

func addToVault(cmd *cobra.Command, vaultPath, encrypted, secretType, dest string, force, dryRun bool) error {
	if dest == "" {
		return fmt.Errorf("--dest is required when using --vault")
	}

	var vault *internalSecrets.Vault

	if path.FileExists(vaultPath) {
		loadedVault, err := internalSecrets.LoadVaultFromFile(vaultPath)
		if err != nil {
			return err
		}
		vault = loadedVault
	} else {
		vault = &internalSecrets.Vault{
			Secrets: []internalSecrets.Secret{},
		}
	}

	newSecret := internalSecrets.Secret{
		Type:        secretType,
		Value:       encrypted,
		Destination: dest,
		Force:       force,
	}

	vault.Secrets = append(vault.Secrets, newSecret)

	yamlData, err := vault.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to marshal vault: %w", err)
	}

	if dryRun {
		fmt.Fprintln(cmd.OutOrStdout(), styles.Warning().Render("[DRY-RUN] Would update vault:"))
		fmt.Fprintln(cmd.OutOrStdout(), string(yamlData))
		return nil
	}

	if err := os.WriteFile(vaultPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write vault file: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n",
		styles.Success().Render(fmt.Sprintf("Secret added to vault %s", vaultPath)))

	return nil
}

func init() {
	encryptCmd.Flags().String("value", "", "Value to encrypt")
	encryptCmd.Flags().String("type", "", "Type of secret (kubeconfig, ssh, env, etc.)")
	encryptCmd.Flags().String("dest", "", "Destination file or environment variable")
	encryptCmd.Flags().String("vault", "", "Path to vault file")
	encryptCmd.Flags().Bool("raw", false, "Output without styling")
}
