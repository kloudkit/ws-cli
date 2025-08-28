package feature

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/apenella/go-ansible/v2/pkg/execute"
	"github.com/apenella/go-ansible/v2/pkg/playbook"
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
	playbookCmd := &playbook.AnsiblePlaybookCmd{
		Playbooks: []string{featurePath},
		PlaybookOptions: &playbook.AnsiblePlaybookOptions{
			ExtraVars: vars,
		},
	}

	exec := execute.NewDefaultExecute(execute.WithCmd(playbookCmd))

	return exec.Execute(context.Background())
}

func installFeatureByName(cmd *cobra.Command, featureName, featuresDir string) error {
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
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n\n", styles.ErrorBadge().Render("ERROR"))
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", styles.Error().Render(err.Error()))
		os.Exit(1)
	}

	return nil
}

func init() {
	installCmd.PersistentFlags().StringToString(
		"opt",
		map[string]string{},
		"Optional variables to use during installation",
	)
}
