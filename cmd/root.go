package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
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

func Execute() {
	ctx := context.Background()

	fangOptions := []fang.Option{
		fang.WithColorSchemeFunc(styles.FrappeColorScheme),
		fang.WithVersion(info.Version),
		fang.WithoutManpage(),
	}

	if err := fang.Execute(ctx, rootCmd, fangOptions...); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(
		clip.ClipCmd,
		feature.FeatureCmd,
		serve.ServeCmd,
		show.ShowCmd,
		template.TemplateCmd,
		info.InfoCmd,
		log.LogCmd,
		logs.LogsCmd,
	)
}
