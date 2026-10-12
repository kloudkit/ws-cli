package serve

import (
	"fmt"

	"github.com/kloudkit/ws-cli/internals/metrics"
	"github.com/kloudkit/ws-cli/internals/server"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:         "metrics",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Start the Prometheus metrics server",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		bind, _ := cmd.Flags().GetString("bind")
		collectors, _ := cmd.Flags().GetStringSlice("collectors")
		out := cmd.OutOrStdout()

		result, err := metrics.BuildRegistry(collectors)
		if err != nil {
			return err
		}

		for _, c := range result.Invalid {
			styles.PrintWarning(out, fmt.Sprintf("Unknown collector '%s', skipping", c))
		}
		for _, w := range result.Warnings {
			styles.PrintWarning(out, w)
		}

		handler := promhttp.HandlerFor(result.Registry, promhttp.HandlerOpts{})

		return server.Serve(server.Config{Port: port, Bind: bind}, handler, out, func(url string) {
			styles.PrintTitle(out, "Metrics Server")
			styles.PrintSubHeader(out, "Collectors")
			styles.PrintBullets(out, result.Expanded)
			fmt.Fprintln(out)
			styles.PrintKeyValues(out, styles.Pair{Key: "URL", Value: url})
		})
	},
}

func init() {
	metricsCmd.Flags().IntP("port", "p", metrics.DefaultPort(), "Port to serve metrics on")
	metricsCmd.Flags().StringSlice("collectors", metrics.DefaultCollectors(), "Comma-separated list of collectors to enable (e.g., workspace,container.cpu,gpu)")

	ServeCmd.AddCommand(metricsCmd)
}
