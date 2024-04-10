package cmd

import (
	"fmt"
	"os"

	"github.com/kloudkit/ws-cli/cmd/feature"
	"github.com/kloudkit/ws-cli/cmd/fonts"
	"github.com/kloudkit/ws-cli/cmd/get"
	"github.com/kloudkit/ws-cli/cmd/log"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "ws-cli",
	Short:   "⚡ CLI companion to charge the workspace batteries",
	Version: "v0.0.1",
	Aliases: []string{"ws"},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(
		feature.FeatureCmd,
		fonts.FontsCmd,
		get.GetCmd,
		log.LogCmd,
	)
}
