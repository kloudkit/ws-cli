package serve

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/kloudkit/ws-cli/cmd/cli"
	"github.com/kloudkit/ws-cli/internals/metrics"
	"github.com/kloudkit/ws-cli/internals/server"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

var currentCmd = cli.New("current", "0.2.0", "Serve current directory as a static site",
	"Serve the current directory over HTTP as a static site — a quick way to preview built files.",
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

var ServeCmd = cli.Group("serve", "0.2.0", "Serve internal assets",
	"Run a small HTTP server for the current directory on --port (default 38080).",
	`# Serve the current directory
ws serve current

# Serve on a custom port
ws serve current --port 38081`,
	currentCmd, metricsCmd)

func serve(cmd *cobra.Command, handler http.Handler, announce func(io.Writer, string)) error {
	config := server.Config{Port: cli.Int(cmd, "port"), Bind: cli.String(cmd, "bind")}

	return server.Serve(config, handler, cmd.OutOrStdout(), func(url string) { announce(cmd.OutOrStdout(), url) })
}

func init() {
	ServeCmd.PersistentFlags().IntP("port", "p", 38080, "Port to serve assets on")
	ServeCmd.PersistentFlags().String("bind", "0.0.0.0", "Bind address")
	metricsCmd.Flags().IntP("port", "p", metrics.DefaultPort(), "Port to serve metrics on")
	metricsCmd.Flags().StringSlice("collectors", metrics.DefaultCollectors(), "Comma-separated list of collectors to enable (e.g., workspace,container.cpu,gpu)")
}
