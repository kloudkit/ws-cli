package feature

import (
	"fmt"

	"github.com/kloudkit/ws-cli/internals/features"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

// skippableSections maps each --skip-* flag to the Ansible extra-var that gates
// the corresponding shared role task. The flags are sugar over the --opt map:
// each lowers to skip_<section>=true, reaching the playbook through the same
// --extra-vars channel as --opt (and the WS_FEATURES_<NAME>_OPTS env channel).
var skippableSections = []struct {
	flag string
	key  string
	desc string
}{
	{"skip-extensions", "skip_extensions", "Skip installing VSCode extensions"},
	{"skip-completion", "skip_completion", "Skip configuring shell completion"},
	{"skip-repository", "skip_repository", "Skip enabling the vendor APT repository"},
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install additional pre-configured features",
	Long:  "Run a feature's playbook to install it. Pass variables with --opt KEY=VAL, and skip parts you do not want with --skip-extensions, --skip-completion, or --skip-repository.",
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

// buildVars assembles the Ansible extra-vars from --opt plus any --skip-* flags.
// An explicit --skip-* flag wins over a colliding --opt skip_<section>=… value.
func buildVars(cmd *cobra.Command) map[string]any {
	rawVars, _ := cmd.Flags().GetStringToString("opt")

	vars := make(map[string]any, len(rawVars)+len(skippableSections))

	for key, value := range rawVars {
		vars[key] = value
	}

	for _, section := range skippableSections {
		if skip, _ := cmd.Flags().GetBool(section.flag); skip {
			vars[section.key] = "true"
		}
	}

	return vars
}

func installFeatureByName(cmd *cobra.Command, featureName, featurePath string) error {
	if _, err := features.ParseFeatureFile(featurePath); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Title().Render(fmt.Sprintf("Installing %s", featureName)))

	if err := features.RunPlaybook(featurePath, buildVars(cmd)); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout())
	styles.PrintSuccessWithDetails(cmd.OutOrStdout(), "Feature installed successfully", [][]string{
		{"Feature", featureName},
	})

	return nil
}

func addInstallFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringToString(
		"opt",
		map[string]string{},
		"Optional variables to use during installation",
	)

	for _, section := range skippableSections {
		cmd.Flags().Bool(section.flag, false, section.desc)
	}
}

func init() {
	addInstallFlags(installCmd)
}
