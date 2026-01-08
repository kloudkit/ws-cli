package secrets

import (
	"fmt"
	"strings"

	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Decrypt a vault spec with encrypted values",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile, _ := cmd.Flags().GetString("input")
		masterKeyFlag, _ := cmd.Flags().GetString("master")
		keys, _ := cmd.Flags().GetStringArray("key")
		force, _ := cmd.Flags().GetBool("force")
		raw, _ := cmd.Flags().GetBool("raw")
		stdout, _ := cmd.Flags().GetBool("stdout")
		modeOverride, _ := cmd.Flags().GetString("mode")

		vaultPath, err := internalSecrets.ResolveVaultPath(inputFile)
		if err != nil {
			return err
		}

		masterKey, err := internalSecrets.ResolveMasterKey(masterKeyFlag)
		if err != nil {
			return err
		}

		vault, err := internalSecrets.LoadVault(vaultPath)
		if err != nil {
			return err
		}

		opts := internalSecrets.ProcessOptions{
			MasterKey:    masterKey,
			Keys:         keys,
			Stdout:       stdout,
			Raw:          raw,
			Force:        force,
			ModeOverride: modeOverride,
		}

		results, err := internalSecrets.ProcessVault(vault, opts)
		if err != nil {
			return err
		}

		if stdout {
			for key, value := range results {
				output := internalSecrets.FormatSecretForStdout(key, value, raw)
				fmt.Fprint(cmd.OutOrStdout(), output)
			}
			return nil
		}

		if !raw {
			fmt.Fprintln(cmd.OutOrStdout(), styles.Success().Render("✓ Vault processed successfully"))
			for key, dest := range results {
				if strings.HasPrefix(dest, "env:") {
					envVar := strings.TrimPrefix(dest, "env:")
					fmt.Fprintf(cmd.OutOrStdout(), "  %s → %s\n",
						styles.Code().Render(key),
						styles.Muted().Render(fmt.Sprintf("env:%s", envVar)))
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s → %s\n",
						styles.Code().Render(key),
						styles.Muted().Render(dest))
				}
			}
		}

		return nil
	},
}

func init() {
	vaultCmd.Flags().String("input", "", "Path to vault file")
	vaultCmd.Flags().StringArray("key", []string{}, "Decrypt only specified key")
	vaultCmd.Flags().Bool("stdout", false, "Output decrypted values to stdout")
}
