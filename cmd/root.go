package cmd

import (
	"context"
	"os"

	"charm.land/fang/v2"
	"github.com/kloudkit/ws-cli/cmd/clip"
	"github.com/kloudkit/ws-cli/cmd/editor"
	"github.com/kloudkit/ws-cli/cmd/feature"
	"github.com/kloudkit/ws-cli/cmd/info"
	"github.com/kloudkit/ws-cli/cmd/log"
	"github.com/kloudkit/ws-cli/cmd/logs"
	"github.com/kloudkit/ws-cli/cmd/secrets"
	"github.com/kloudkit/ws-cli/cmd/seed"
	"github.com/kloudkit/ws-cli/cmd/serve"
	"github.com/kloudkit/ws-cli/cmd/show"
	"github.com/kloudkit/ws-cli/cmd/template"
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "ws-cli",
	Short:         "⚡ CLI companion to charge the workspace batteries",
	Long:          "The workspace command-line companion. Groups helpers for inspecting the running workspace, managing settings and secrets, driving the editor, and serving local assets — most are called by the startup scripts, all are yours in the terminal.",
	Version:       info.Version(),
	Aliases:       []string{"ws"},
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return config.Bootstrap()
	},
}

func RootCmd() *cobra.Command {
	return rootCmd
}

func Execute() {
	ctx := context.Background()

	fangOptions := []fang.Option{
		fang.WithColorSchemeFunc(styles.FrappeColorScheme),
		fang.WithVersion(info.Version()),
		fang.WithoutManpage(),
	}

	if err := fang.Execute(ctx, rootCmd, fangOptions...); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(
		clip.ClipCmd,
		editor.EditorCmd,
		feature.FeatureCmd,
		serve.ServeCmd,
		show.ShowCmd,
		template.TemplateCmd,
		info.InfoCmd,
		log.LogCmd,
		logs.LogsCmd,
		secrets.SecretsCmd,
		seed.SeedCmd,
	)
}
