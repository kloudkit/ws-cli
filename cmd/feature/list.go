package feature

import (
	"fmt"
	"strings"

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
			fmt.Fprintln(cmd.OutOrStdout(), styles.Warning().Render("No features found"))
			return nil
		}

		fmt.Fprintf(
			cmd.OutOrStdout(),
			"\n%s\n\n",
			styles.InfoBadge().Render(fmt.Sprintf("%d FEATURES AVAILABLE", len(availableFeatures))),
		)

		featuresTable := styles.Table("NAME", "DESCRIPTION", "OPTIONS")

		for _, feature := range availableFeatures {
			var options string
			if len(feature.Vars) > 0 {
				var styledVars []string
				for _, v := range feature.Vars {
					styledVars = append(styledVars, styles.Key().UnsetBold().Render(v))
				}
				options = strings.Join(styledVars, ", ")
			} else {
				options = styles.Muted().Render("-")
			}

			featuresTable.Row(feature.Name, feature.Description, options)
		}

		fmt.Fprintln(cmd.OutOrStdout(), featuresTable.String())

		return nil
	},
}
