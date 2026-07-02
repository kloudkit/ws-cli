package feature

import (
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/spf13/cobra"
)

var FeatureCmd = &cobra.Command{
	Use:         "feature",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Install additional pre-configured features",
	Long:        "Install and inspect optional workspace features — Ansible playbooks that add tools on top of the base image. Ships a curated set; --root points at your own under ~/.ws/features.d.",
}

func featureDirs(cmd *cobra.Command) []string {
	root, _ := cmd.Flags().GetString("root")

	if cmd.Flags().Changed("root") {
		return []string{root}
	}

	return []string{root, path.GetHomeDirectory(".ws", "features.d")}
}

func init() {
	root, _ := config.Resolve("features", "dir")

	FeatureCmd.PersistentFlags().String(
		"root",
		root,
		"Root directory of additional features",
	)

	FeatureCmd.AddCommand(installCmd, listCmd, infoCmd, storeCmd, newCmd)
}
