package feature

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"

	"github.com/kloudkit/ws-cli/internals/features"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install additional pre-configured features",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		featuresDir, _ := cmd.Flags().GetString("root")
		featureName := args[0]

		result, err := features.ListFeatures(featuresDir)
		if err != nil {
			return fmt.Errorf("failed to list features: %w", err)
		}

		featureExists := slices.ContainsFunc(result.Features, func(f *features.Feature) bool {
			return f.Name == featureName
		})

		if !featureExists {
			return fmt.Errorf("feature '%s' not found", featureName)
		}

		return installFeatureByName(cmd, featureName, featuresDir)
	},
}

func installFeatureByName(cmd *cobra.Command, featureName, featuresDir string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Title().Render(fmt.Sprintf("Installing %s", featureName)))

	rawVars, _ := cmd.Flags().GetStringToString("opt")

	vars := make(map[string]any, len(rawVars))

	keys := make([]string, 0, len(rawVars))
	for key := range rawVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		vars[key] = rawVars[key]
	}

	featurePath := filepath.Join(featuresDir, featureName+".yaml")

	if err := features.RunPlaybook(featurePath, vars); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout())
	styles.PrintSuccessWithDetails(cmd.OutOrStdout(), "Feature installed successfully", [][]string{
		{"Feature", featureName},
	})

	return nil
}

func init() {
	installCmd.PersistentFlags().StringToString(
		"opt",
		map[string]string{},
		"Optional variables to use during installation",
	)
}
