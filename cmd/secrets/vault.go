package secrets

import (
	"fmt"

	"github.com/kloudkit/ws-cli/internals/path"
	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/spf13/cobra"
)

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Create an encrypted vault",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := newContext(cmd)
		input := getString(cmd, "input")
		output := getString(cmd, "output")

		if !path.FileExists(input) {
			return fmt.Errorf("input file not found: %s", input)
		}

		if output != "" && !path.CanOverride(output, ctx.force) {
			return fmt.Errorf("output file %s exists, use --force to overwrite", output)
		}

		vault, err := internalSecrets.LoadVaultFromFile(input)
		if err != nil {
			return err
		}

		masterKey, err := ctx.resolveMasterKey()
		if err != nil {
			return err
		}

		if err := vault.EncryptAll(masterKey); err != nil {
			return err
		}

		if ctx.dryRun {
			yamlData, _ := vault.ToYAML()
			ctx.dryRunMsg("[DRY-RUN] Would write encrypted vault:")
			fmt.Fprintln(cmd.OutOrStdout(), string(yamlData))
			return nil
		}

		if output == "" {
			yamlData, _ := vault.ToYAML()
			fmt.Fprint(cmd.OutOrStdout(), string(yamlData))
		} else {
			if err := vault.SaveToFile(output); err != nil {
				return err
			}
			ctx.success(fmt.Sprintf("Encrypted vault written to %s", output))
		}

		return nil
	},
}

func init() {
	vaultCmd.Flags().String("input", "", "Input plain YAML file")
	vaultCmd.Flags().String("output", "", "Output encrypted vault file (default stdout)")

	vaultCmd.MarkFlagRequired("input")
}
