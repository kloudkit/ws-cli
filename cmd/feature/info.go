package feature

import (
	"fmt"

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

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Title().Render(feature.Name))
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", styles.Value().Render(feature.Description))

		if len(feature.Vars) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", styles.Header().Render("Options"))

			listItems := make([]any, len(feature.Vars))
			for i, v := range feature.Vars {
				listItems[i] = styles.Key().UnsetBold().Render(v)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.List(listItems...))
		}

		return nil
	},
}
