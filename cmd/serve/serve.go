package serve

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kloudkit/ws-cli/cmd/cli"
	"github.com/kloudkit/ws-cli/internals/mcp"
	"github.com/kloudkit/ws-cli/internals/metrics"
	"github.com/kloudkit/ws-cli/internals/server"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

var currentCmd = cli.New("current", "0.2.0", "Serve current directory as a static site",
	"Serve the current directory over HTTP as a static site, a quick way to preview built files.",
	func(cmd *cobra.Command, args []string) error {
		root, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current directory: %w", err)
		}

		return serve(cmd, http.FileServer(http.Dir(root)), func(out io.Writer, url string) {
			styles.PrintTitle(out, "Static Server")
			styles.PrintKeyValues(out, styles.Pair{Key: "URL", Value: url}, styles.Pair{Key: "Root", Value: root})
		})
	})

var metricsCmd = cli.New("metrics", "0.2.0", "Start the Prometheus metrics server",
	"Serve workspace and container metrics in the Prometheus exposition format, from the collectors chosen with --collectors.",
	func(cmd *cobra.Command, args []string) error {
		collectors, _ := cmd.Flags().GetStringSlice("collectors")

		result, err := metrics.BuildRegistry(collectors)
		if err != nil {
			return err
		}

		for _, c := range result.Invalid {
			styles.PrintWarning(cmd.OutOrStdout(), fmt.Sprintf("Unknown collector '%s', skipping", c))
		}
		for _, w := range result.Warnings {
			styles.PrintWarning(cmd.OutOrStdout(), w)
		}

		return serve(cmd, promhttp.HandlerFor(result.Registry, promhttp.HandlerOpts{}), func(out io.Writer, url string) {
			styles.PrintTitle(out, "Metrics Server")
			styles.PrintKeyValues(out, styles.Pair{Key: "URL", Value: url})
			fmt.Fprintln(out)
			styles.PrintSubHeader(out, "Collectors")
			styles.PrintBullets(out, result.Expanded)
		})
	})

var mcpCmd = cli.New("mcp", "next", "Serve the workspace tools to AI agents over MCP",
	"Run a Model Context Protocol server on stdin and stdout, for an AI agent in the workspace to start. Its tools are the ws-cli commands that opt in: read-only ones by default, the ones that act on the workspace too with WS_AI_MCP_ALLOW_WRITE, narrowed to the groups or tools named in WS_AI_MCP_TOOLS.",
	func(cmd *cobra.Command, args []string) error {
		settings, warnings := mcp.ResolveSettings()

		tools, unmatched, err := mcp.BuildTools(cmd.Root(), settings)
		if err != nil {
			return err
		}

		for _, warning := range append(warnings, unmatched...) {
			styles.PrintWarning(cmd.ErrOrStderr(), warning)
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
		defer stop()
		context.AfterFunc(ctx, stop)

		return mcp.Serve(ctx, cmd.Root().Version, tools)
	})

var ServeCmd = cli.Group("serve", "0.2.0", "Serve internal assets",
	"Run a small local server: the current directory as a static site, the Prometheus metrics, or the workspace tools for AI agents over MCP.",
	`# Serve the current directory
ws-cli serve current

# Serve on a custom port
ws-cli serve current --port 38081

# Serve the workspace tools to an AI agent over stdio
ws-cli serve mcp`,
	currentCmd, mcpCmd, metricsCmd)

func serve(cmd *cobra.Command, handler http.Handler, announce func(io.Writer, string)) error {
	config := server.Config{Port: cli.Int(cmd, "port"), Bind: cli.String(cmd, "bind")}

	return server.Serve(config, handler, cmd.OutOrStdout(), func(url string) { announce(cmd.OutOrStdout(), url) })
}

func init() {
	currentCmd.Flags().IntP("port", "p", 38080, "Port to serve assets on")
	currentCmd.Flags().String("bind", "0.0.0.0", "Bind address")
	mcpCmd.Args = cobra.NoArgs
	metricsCmd.Flags().String("bind", "0.0.0.0", "Bind address")
	metricsCmd.Flags().IntP("port", "p", metrics.DefaultPort(), "Port to serve metrics on")
	metricsCmd.Flags().StringSlice("collectors", metrics.DefaultCollectors(), "Comma-separated list of collectors to enable (e.g., workspace,container.cpu,gpu)")
}
