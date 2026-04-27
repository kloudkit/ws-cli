package feature

import (
	"fmt"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/kloudkit/ws-cli/internals/features"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "List packages available in the feature store",
	RunE: func(cmd *cobra.Command, args []string) error {
		storeURL := env.String(config.EnvFeaturesStoreURL)
		if storeURL == "" {
			styles.PrintWarning(cmd.OutOrStdout(), "Feature store not configured (set WS_FEATURES_STORE_URL)")
			return nil
		}

		manifest, err := features.FetchStoreManifest(storeURL)
		if err != nil {
			return fmt.Errorf("failed to fetch store manifest: %w", err)
		}

		if len(manifest.Artifacts) == 0 {
			styles.PrintWarning(cmd.OutOrStdout(), "Feature store is empty")
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.TitleWithCount("Feature Store", len(manifest.Artifacts)))

		items := make([]styles.DescriptionItem, len(manifest.Artifacts))
		for i, a := range manifest.Artifacts {
			items[i] = styles.DescriptionItem{Name: a.Name, Description: a.Version}
		}

		fmt.Fprintln(cmd.OutOrStdout(), styles.List(styles.DescriptionList(items)...))

		if manifest.Built != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "\n  %s\n", styles.Muted().Render("Built: "+manifest.Built))
		}

		return nil
	},
}
