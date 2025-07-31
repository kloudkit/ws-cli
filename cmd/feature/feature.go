package feature

import (
  "os"

	"github.com/kloudkit/ws-cli/cmd/feature/install"
	"github.com/spf13/cobra"
)

var FeatureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Install extra pre-configured features",
}

func init() {
  rootDir := os.Getenv("WS_FEATURES_DIR")

  if rootDir == "" {
		rootDir = "/features"
	}

	FeatureCmd.PersistentFlags().String("root", rootDir, "Root directory of extra features")

	FeatureCmd.AddCommand(install.InstallCmd)
}
