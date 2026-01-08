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

		availableFeatures, err := features.ListFeatures(featuresDir)
		if err != nil {
			return fmt.Errorf("failed to list features: %w", err)
		}

		if len(availableFeatures) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), styles.Warning().Render("⚠ No features found"))
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.TitleWithCount("Features Available", len(availableFeatures)))

		maxNameLen := 0
		for _, feature := range availableFeatures {
			if len(feature.Name) > maxNameLen {
				maxNameLen = len(feature.Name)
			}
		}

		var featureItems []any
		for _, feature := range availableFeatures {
			item := styles.Key().Width(maxNameLen).Render(feature.Name) +
				styles.Muted().Render(" — ") +
				styles.Value().Render(feature.Description)

			featureItems = append(featureItems, item)
		}

		fmt.Fprintln(cmd.OutOrStdout(), styles.List(featureItems...))

		styles.PrintHints(cmd.OutOrStdout(), [][]string{
			{"ws-cli feature install <name>", "Install a feature"},
			{"ws-cli feature info <name>", "View feature details"},
		})

		return nil
	},
}
