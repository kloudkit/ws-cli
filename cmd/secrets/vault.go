package secrets

import (
	"fmt"
	"os"

	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Create an encrypted vault",
	RunE: func(cmd *cobra.Command, args []string) error {
		input, _ := cmd.Flags().GetString("input")
		output, _ := cmd.Flags().GetString("output")
		force, _ := cmd.Flags().GetBool("force")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		masterKey, _ := cmd.Flags().GetString("master")

		if !path.FileExists(input) {
			return fmt.Errorf("input file not found: %s", input)
		}

		if output != "" && !path.CanOverride(output, force) {
			return fmt.Errorf("output file %s exists, use --force to overwrite", output)
		}

		vault, err := internalSecrets.LoadVaultFromFile(input)
		if err != nil {
			return err
		}

		key, err := internalSecrets.ResolveMasterKey(masterKey)
		if err != nil {
			return err
		}

		if err := vault.EncryptAll(key); err != nil {
			return err
		}

		yamlData, err := vault.ToYAML()
		if err != nil {
			return fmt.Errorf("failed to marshal vault: %w", err)
		}

		if dryRun {
			fmt.Fprintln(cmd.OutOrStdout(), styles.Warning().Render("[DRY-RUN] Would write encrypted vault:"))
			fmt.Fprintln(cmd.OutOrStdout(), string(yamlData))
			return nil
		}

		if output == "" {
			fmt.Fprint(cmd.OutOrStdout(), string(yamlData))
		} else {
			if err := os.WriteFile(output, yamlData, 0644); err != nil {
				return fmt.Errorf("failed to write vault file: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n",
				styles.Success().Render(fmt.Sprintf("Encrypted vault written to %s", output)))
		}

		return nil
	},
}

func init() {
	vaultCmd.Flags().String("input", "", "Input plain YAML file")
	vaultCmd.Flags().String("output", "", "Output encrypted vault file (default stdout)")

	vaultCmd.MarkFlagRequired("input")
}
