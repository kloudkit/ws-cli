package feature

import (
	"fmt"
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
		featureName := args[0]

		featurePath, err := features.ResolveFeaturePath(featureDirs(cmd), featureName)
		if err != nil {
			return err
		}

		return installFeatureByName(cmd, featureName, featurePath)
	},
}

func installFeatureByName(cmd *cobra.Command, featureName, featurePath string) error {
	if _, err := features.ParseFeatureFile(featurePath); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

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
