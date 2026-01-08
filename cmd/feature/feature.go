package feature

import (
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/spf13/cobra"
)

var FeatureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Install additional pre-configured features",
}

func init() {
	FeatureCmd.PersistentFlags().String(
		"root",
		env.String(config.EnvFeaturesDir, config.DefaultFeaturesDir),
		"Root directory of additional features",
	)

	FeatureCmd.AddCommand(installCmd, listCmd, infoCmd)
}
