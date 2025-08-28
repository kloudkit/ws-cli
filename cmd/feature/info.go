package feature

import (
	"fmt"
	"strings"

	"github.com/kloudkit/ws-cli/internals/features"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show detailed information about a feature",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		featuresDir, _ := cmd.Flags().GetString("root")
		featureName := args[0]

		feature, err := features.InfoFeature(featuresDir, featureName)
		if err != nil {
			return fmt.Errorf("failed to get feature info: %w", err)
		}

		fmt.Fprintf(
			cmd.OutOrStdout(),
			"\n%s\n  %s\n\n",
			styles.InfoBadge().Render(strings.ToUpper(feature.Name)),
			feature.Description,
		)

		if len(feature.Vars) > 0 {
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"%s\n",
				styles.Badge().Render("Options"),
			)

			listItems := make([]any, len(feature.Vars))
			for i, v := range feature.Vars {
				listItems[i] = styles.Key().UnsetBold().Render(v)
			}

			fmt.Fprintln(cmd.OutOrStdout(), styles.List(listItems...))
		}

		fmt.Fprintln(cmd.OutOrStdout())
		return nil
	},
}
