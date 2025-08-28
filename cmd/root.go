package cmd

import (
	"fmt"
	"os"

	"github.com/kloudkit/ws-cli/cmd/clip"
	"github.com/kloudkit/ws-cli/cmd/config"
	"github.com/kloudkit/ws-cli/cmd/feature"
	"github.com/kloudkit/ws-cli/cmd/get"
	"github.com/kloudkit/ws-cli/cmd/info"
	"github.com/kloudkit/ws-cli/cmd/log"
	"github.com/kloudkit/ws-cli/cmd/logs"
	"github.com/kloudkit/ws-cli/cmd/serve"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "ws-cli",
	Short:   "⚡ CLI companion to charge the workspace batteries",
	Version: "v" + info.Version,
	Aliases: []string{"ws"},
}

var noColor bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display installed workspace version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), info.Version)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color output")

	cobra.OnInitialize(func() {
		if _, exists := os.LookupEnv("WS_LOGGING_NO_COLOR"); exists {
			styles.ColorEnabled = false
		}

		if noColor {
			styles.ColorEnabled = false
		}
	})

	rootCmd.AddCommand(
		clip.ClipCmd,
		config.ConfigCmd,
		feature.FeatureCmd,
		serve.ServeCmd,
		get.GetCmd,
		info.InfoCmd,
		log.LogCmd,
		logs.LogsCmd,
		versionCmd,
	)
}
