package secrets

import (
	"fmt"

	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/spf13/cobra"
)

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Create an encrypted vault",
	RunE: func(cmd *cobra.Command, args []string) error {
		input, _ := cmd.Flags().GetString("input")
		output, _ := cmd.Flags().GetString("output")
		force, _ := cmd.Flags().GetBool("force")

		if !path.FileExists(input) {
			return fmt.Errorf("input file not found: %s", input)
		}

		if output != "" && !path.CanOverride(output, force) {
			return fmt.Errorf("output file %s exists, use --force to overwrite", output)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Vault creation logic not yet implemented")

		return nil
	},
}

func init() {
	vaultCmd.Flags().String("input", "", "Input plain YAML file")
	vaultCmd.Flags().String("output", "", "Output encrypted vault file (default stdout)")

	vaultCmd.MarkFlagRequired("input")
}
