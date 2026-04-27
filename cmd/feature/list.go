package feature

import (
	"fmt"

	"github.com/kloudkit/ws-cli/internals/features"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List available features that can be installed",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		featuresDir, _ := cmd.Flags().GetString("root")

		result, err := features.ListFeatures(featuresDir)
		if err != nil {
			return fmt.Errorf("failed to list features: %w", err)
		}

		for _, w := range result.Warnings {
			styles.PrintWarning(cmd.OutOrStdout(), w)
		}

		if len(result.Features) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), styles.Warning().Render("⚠ No features found"))
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.TitleWithCount("Features Available", len(result.Features)))

		items := make([]styles.DescriptionItem, len(result.Features))
		for i, f := range result.Features {
			items[i] = styles.DescriptionItem{Name: f.Name, Description: f.Description}
		}

		fmt.Fprintln(cmd.OutOrStdout(), styles.List(styles.DescriptionList(items)...))

		styles.PrintHints(cmd.OutOrStdout(), [][]string{
			{"ws-cli feature install <name>", "Install a feature"},
			{"ws-cli feature info <name>", "View feature details"},
		})

		return nil
	},
}
