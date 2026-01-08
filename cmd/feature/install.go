package feature

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

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

		availableFeatures, err := features.ListFeatures(featuresDir)

		if err != nil {
			return fmt.Errorf("failed to list features: %w", err)
		}

		featureExists := slices.ContainsFunc(availableFeatures, func(f *features.Feature) bool {
			return f.Name == featureName
		})

		if !featureExists {
			return fmt.Errorf("feature '%s' not found", featureName)
		}

		return installFeatureByName(cmd, featureName, featuresDir)
	},
}

func runAnsiblePlaybook(featurePath string, vars map[string]any) error {
	args := []string{featurePath}

	if len(vars) > 0 {
		var extraVars []string
		for key, value := range vars {
			extraVars = append(extraVars, fmt.Sprintf("%s=%v", key, value))
		}
		args = append(args, "--extra-vars", strings.Join(extraVars, " "))
	}

	cmd := exec.Command("ansible-playbook", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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

	if _, err := features.InfoFeature(featuresDir, featureName); err != nil {
		return fmt.Errorf("feature installation failed: %w", err)
	}

	if err := runAnsiblePlaybook(featurePath, vars); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr())
		styles.PrintError(cmd.ErrOrStderr(), err.Error())
		os.Exit(1)
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
