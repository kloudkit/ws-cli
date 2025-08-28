package feature

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apenella/go-ansible/v2/pkg/execute"
	"github.com/apenella/go-ansible/v2/pkg/playbook"
	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/kloudkit/ws-cli/internals/features"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install additional pre-configured features",
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

func installFeature(featureName string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		root, _ := cmd.Flags().GetString("root")
		rawVars, _ := cmd.Flags().GetStringToString("opt")

		vars := make(map[string]any, len(rawVars))
		for key, value := range rawVars {
			vars[key] = value
		}

		featurePath := filepath.Join(root, featureName+".yaml")

		if _, err := features.InfoFeature(root, featureName); err != nil {
			return fmt.Errorf("feature installation failed: %w", err)
		}

		if err := runAnsiblePlaybook(featurePath, vars); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s\n\n", styles.ErrorBadge().Render("ERROR"))
			fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", styles.Error().Render(err.Error()))
			os.Exit(1)
		}

		return nil
	}
}

func init() {
	installCmd.PersistentFlags().StringToString(
		"opt",
		map[string]string{},
		"Optional variables to use during installation",
	)

	availableFeatures, err := features.ListFeatures(env.String("WS_FEATURES_DIR", "/features"))

	if err == nil {
		for _, feature := range availableFeatures {
			installCmd.AddCommand(&cobra.Command{
				Use:   feature.Name,
				Short: feature.Description,
				RunE:  installFeature(feature.Name),
			})
		}
	}
}
