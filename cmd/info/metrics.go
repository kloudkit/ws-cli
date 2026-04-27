package info

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kloudkit/ws-cli/internals/metrics"
	"github.com/kloudkit/ws-cli/internals/styles"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Display workspace metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		includeGPU, _ := cmd.Flags().GetBool("gpu")

		m, err := metrics.GetWorkspaceSummary(includeGPU)
		if err != nil {
			styles.PrintWarning(cmd.OutOrStdout(), "Could not read workspace metrics")
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Title().Render("Metrics"))

		rows := [][]string{
			{"CPU", fmt.Sprintf("%.1f%% (%s)", m.CPUUsage, styles.FormatCPUTime(m.CPUSeconds))},
			{"Memory", fmt.Sprintf("%s / %s (%s)",
				styles.FormatBytes(m.MemoryUsed),
				styles.FormatBytes(m.MemoryTotal),
				styles.FormatPercent(m.MemoryUsed, m.MemoryTotal))},
			{"Memory RSS", styles.FormatBytes(m.MemoryRSS)},
			{"Disk", fmt.Sprintf("%s / %s (%s)",
				styles.FormatBytes(m.DiskUsed),
				styles.FormatBytes(m.DiskTotal),
				styles.FormatPercent(m.DiskUsed, m.DiskTotal))},
			{"File Descriptors", fmt.Sprintf("%d / %d", m.FDOpen, m.FDLimit)},
		}

		if m.GPU != nil && m.GPU.Available {
			rows = append(rows,
				[]string{"GPU Utilization", fmt.Sprintf("%.0f%%", m.GPU.UtilizationRatio*100)},
				[]string{"GPU Memory", fmt.Sprintf("%s / %s",
					styles.FormatBytes(m.GPU.MemoryUsedBytes),
					styles.FormatBytes(m.GPU.MemoryTotalBytes))},
				[]string{"GPU Temperature", fmt.Sprintf("%.0fC", m.GPU.TemperatureCelsius)},
				[]string{"GPU Power", fmt.Sprintf("%.1fW", m.GPU.PowerWatts)},
			)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Table().Rows(rows...).Render())

		return nil
	},
}

func init() {
	metricsCmd.Flags().Bool("gpu", false, "Include GPU metrics")
	InfoCmd.AddCommand(metricsCmd)
}
