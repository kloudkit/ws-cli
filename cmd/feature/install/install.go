package install

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/apenella/go-ansible/v2/pkg/execute"
	"github.com/apenella/go-ansible/v2/pkg/playbook"
	"github.com/spf13/cobra"
)

var InstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install extra pre-configured features",
}

var features = map[string]string{
	"cloudflared": "Install Cloudflare tunnel CLI",
	"conan":       "Install Conan CLI and related tools",
  "cpp":         "Install C++ and related tools",
	"dagger":      "Install dagger.io CLI and SDK",
	"dotnet":      "Install .NET framework and related extensions",
	"gcloud":      "Install Google Cloud CLI for GCP",
	"gh":          "Install GitHub CLI",
	"jf":          "Install JFrog CLI",
	"jupyter":     "Install Jupyter packages and related extensions",
	"php":         "Install PHP and related extensions",
	"rclone":      "Install rclone CLI",
	"restic":      "Install Restic CLI",
	"rust":        "Install Rust and Cargo",
	"terraform":   "Install Terraform and related extensions",
}

var customCmd = &cobra.Command{
	Use:   "custom",
	Short: "Install a custom feature",
	Run:   install("custom"),
}

func runPlay(feature string, vars map[string]interface{}, errorOut io.Writer) {
	playbookCmd := &playbook.AnsiblePlaybookCmd{
		Playbooks: []string{feature},
		PlaybookOptions: &playbook.AnsiblePlaybookOptions{
			ExtraVars: vars,
		},
	}

	exec := execute.NewDefaultExecute(
		execute.WithCmd(playbookCmd),
	)

	err := exec.Execute(context.Background())

	if err != nil {
		fmt.Fprintln(errorOut, err)
		os.Exit(1)
	}
}

func getFeaturePath(root string, feature string, errorOut io.Writer) string {
	feature = filepath.Join(root, feature+".yaml")

	if _, err := os.Stat(feature); os.IsNotExist(err) {
		fmt.Fprintf(errorOut, "ERROR: The feature path [%s] could not be found.\n", feature)
		os.Exit(1)
	}

	return feature
}

func install(feature string) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		root, _ := cmd.Flags().GetString("root")
		rawVars, _ := cmd.Flags().GetStringToString("opt")

		vars := make(map[string]interface{})

		for key, value := range rawVars {
			vars[key] = value
		}

		if feature == "custom" {
			customFeature, _ := cmd.Flags().GetString("feature")

			feature = customFeature
		}

		feature = getFeaturePath(root, feature, cmd.ErrOrStderr())

		runPlay(feature, vars, cmd.ErrOrStderr())
	}
}

func init() {
	InstallCmd.PersistentFlags().StringToString(
		"opt",
		map[string]string{},
		"Optional variables to use during installation",
	)

	customCmd.Flags().String("feature", "", "The custom feature to install")
	customCmd.MarkFlagRequired("feature")

	InstallCmd.AddCommand(customCmd)

	for feature, description := range features {
		InstallCmd.AddCommand(&cobra.Command{
			Use:   feature,
			Short: description,
			Run:   install(feature),
		})
	}
}
