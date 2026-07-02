package feature

import (
	"fmt"

	"github.com/kloudkit/ws-cli/internals/features"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:         "list",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "List available features that can be installed",
	Long:        "List the features you can install, marking where each comes from — the shipped set, a workspace override, or your own ~/.ws/features.d.",
	Aliases:     []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := features.ListFeatures(featureDirs(cmd))
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
			items[i] = styles.DescriptionItem{Name: f.Name, Description: describeSource(f)}
		}

		fmt.Fprintln(cmd.OutOrStdout(), styles.List(styles.DescriptionList(items)...))

		styles.PrintHints(cmd.OutOrStdout(), [][]string{
			{"ws-cli feature install <name>", "Install a feature"},
			{"ws-cli feature info <name>", "View feature details"},
		})

		return nil
	},
}

func describeSource(f *features.Feature) string {
	switch f.Source {
	case features.SourceUser:
		return f.Description + " (user)"
	case features.SourceOverride:
		return f.Description + " (override)"
	default:
		return f.Description
	}
}
