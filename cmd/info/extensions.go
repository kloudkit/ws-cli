package info

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/styles"
)

var extensionsCmd = &cobra.Command{
	Use:         "extensions",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Display installed extensions",
	Long:        "List the installed VS Code extensions with their versions.",
	RunE: func(cmd *cobra.Command, args []string) error {
		extensions, _ := config.GetExtensions()

		var rows [][]string
		for _, ext := range extensions {
			rows = append(rows, []string{ext.Name, ext.Version})
		}

		if len(extensions) == 0 {
			styles.PrintEmpty(cmd.OutOrStdout(), "No extensions installed")
			return nil
		}

		styles.PrintTitleCount(cmd.OutOrStdout(), "Extensions", len(extensions))
		fmt.Fprintln(cmd.OutOrStdout(), styles.NewTable("Name", "Version").Rows(rows...).Render())

		return nil
	},
}

func init() {
	InfoCmd.AddCommand(extensionsCmd)
}
