package feature

import (
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/spf13/cobra"
)

var FeatureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Install additional pre-configured features",
}

func init() {
	root, _ := config.Resolve("features", "dir")

	FeatureCmd.PersistentFlags().String(
		"root",
		root,
		"Root directory of additional features",
	)

	FeatureCmd.AddCommand(installCmd, listCmd, infoCmd, storeCmd)
}
