package vault

import (
	"fmt"

	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List secrets in a vault",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile, _ := cmd.Flags().GetString("input")
		raw, _ := cmd.Flags().GetBool("raw")

		vaultPath, err := internalSecrets.ResolveVaultPath(inputFile)
		if err != nil {
			return err
		}

		vault, err := internalSecrets.LoadRawVault(vaultPath)
		if err != nil {
			return err
		}

		entries := internalSecrets.ListVault(vault)
		if len(entries) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), styles.Muted().Render("No secrets in vault"))
			return nil
		}

		if raw {
			for _, e := range entries {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", e.Name, e.Type, e.Destination)
			}
			return nil
		}

		t := styles.Table("Name", "Type", "Destination")
		for _, e := range entries {
			t.Row(e.Name, e.Type, e.Destination)
		}
		fmt.Fprintln(cmd.OutOrStdout(), t.Render())

		return nil
	},
}
