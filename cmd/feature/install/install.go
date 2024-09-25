package install

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/apenella/go-ansible/v2/pkg/execute"
	"github.com/apenella/go-ansible/v2/pkg/playbook"
	"github.com/spf13/cobra"
)

var InstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install extra pre-configured features",
}

var conanCmd = &cobra.Command{
	Use:   "conan",
	Short: "Install conan CLI and related tools",
	Run:   install("conan"),
}

var daggerCmd = &cobra.Command{
	Use:   "dagger",
	Short: "Install dagger.io CLI and SDK",
	Run:   install("dagger"),
}

var gcloudCmd = &cobra.Command{
	Use:   "gcloud",
	Short: "Install gcloud CLI for GCP",
	Run:   install("gcloud"),
}

var jupyterCmd = &cobra.Command{
	Use:   "jupyter",
	Short: "Install Jupyter packages and related extensions",
	Run:   install("jupyter"),
}

var dotnetCmd = &cobra.Command{
	Use:   "dotnet",
	Short: "Install the .NET framework and related extensions",
	Run:   install("dotnet"),
}

var phpCmd = &cobra.Command{
	Use:   "php",
	Short: "Install PHP and related extensions",
	Run:   install("php"),
}

var resticCmd = &cobra.Command{
	Use:   "restic",
	Short: "Install Restic",
	Run:   install("restic"),
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
	if !strings.HasSuffix(root, "/") {
		root += "/"
	}

	feature = strings.Join([]string{root, feature, ".yaml"}, "")

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

	InstallCmd.AddCommand(
		conanCmd,
		daggerCmd,
		dotnetCmd,
		gcloudCmd,
		jupyterCmd,
		phpCmd,
    resticCmd,
  )
}
