package cmd

import (
	"fmt"
	"os"

	"github.com/kloudkit/ws-cli/cmd/clip"
	"github.com/kloudkit/ws-cli/cmd/feature"
	"github.com/kloudkit/ws-cli/cmd/info"
	"github.com/kloudkit/ws-cli/cmd/log"
	"github.com/kloudkit/ws-cli/cmd/logs"
	"github.com/kloudkit/ws-cli/cmd/serve"
	"github.com/kloudkit/ws-cli/cmd/show"
	"github.com/kloudkit/ws-cli/cmd/template"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "ws-cli",
	Short:         "⚡ CLI companion to charge the workspace batteries",
	Version:       "v" + info.Version,
	Aliases:       []string{"ws"},
	SilenceErrors: true,
}

var noColor bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the installed workspace version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), styles.Value().Render(info.Version))
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {

		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.ErrorBadge().Render("ERROR"),
			styles.Error().Render(err.Error()),
		)

		// Add usage hint for command resolution errors
		// if strings.Contains(err.Error(), "unknown command") {
		// 	fmt.Fprintf(os.Stderr, "Run '%s --help' for usage.\n", rootCmd.Use)
		// }

		// Print the original error again to maintain the expected behavior
		// fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())

		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color output")

	rootCmd.SetHelpTemplate(styles.HelpTemplate())
	rootCmd.SetUsageTemplate(styles.UsageTemplate())
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return err
	})

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
		feature.FeatureCmd,
		serve.ServeCmd,
		show.ShowCmd,
		template.TemplateCmd,
		info.InfoCmd,
		log.LogCmd,
		logs.LogsCmd,
		versionCmd,
	)
}
