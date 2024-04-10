package feature

import (
	"github.com/kloudkit/ws-cli/cmd/feature/install"
	"github.com/spf13/cobra"
)

var FeatureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Install extra pre-configured features",
}

func init() {
	FeatureCmd.PersistentFlags().String("root", "/features", "Root directory of extra features")

	FeatureCmd.AddCommand(install.InstallCmd)
}
