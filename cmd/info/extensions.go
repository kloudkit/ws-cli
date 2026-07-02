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

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.TitleWithCount("Extensions", len(extensions)))

		t := styles.Table("Name", "Version").Rows(rows...)

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", t.Render())

		return nil
	},
}

func init() {
	InfoCmd.AddCommand(extensionsCmd)
}
