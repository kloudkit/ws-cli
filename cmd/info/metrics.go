package info

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kloudkit/ws-cli/internals/metrics"
	"github.com/kloudkit/ws-cli/internals/styles"
)

var metricsCmd = &cobra.Command{
	Use:         "metrics",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Display workspace metrics",
	Long:        "Show live resource usage — CPU, memory, disk, and file descriptors, plus GPU with --gpu.",
	RunE: func(cmd *cobra.Command, args []string) error {
		includeGPU, _ := cmd.Flags().GetBool("gpu")

		m, err := metrics.GetWorkspaceSummary(includeGPU)
		if err != nil {
			styles.PrintWarning(cmd.OutOrStdout(), "Could not read workspace metrics")
			return nil
		}

		styles.PrintTitle(cmd.OutOrStdout(), "Metrics")

		rows := [][]string{
			{"CPU", fmt.Sprintf("%.1f%% · %s CPU time", m.CPUUsage, styles.FormatCPUTime(m.CPUSeconds))},
			{"Memory", formatMemory(m.MemoryUsed, m.MemoryTotal)},
			{"Memory RSS", styles.FormatBytes(m.MemoryRSS)},
			{"Disk", fmt.Sprintf("%s / %s (%s)",
				styles.FormatBytes(m.DiskUsed),
				styles.FormatBytes(m.DiskTotal),
				styles.FormatPercent(m.DiskUsed, m.DiskTotal))},
			{"File Descriptors", fmt.Sprintf("%s open · limit %s per process", styles.FormatCount(m.FDOpen), styles.FormatCount(m.FDLimit))},
		}

		switch {
		case !includeGPU:
		case m.GPU == nil || !m.GPU.Available:
			rows = append(rows, []string{"GPU", "not available"})
		default:
			rows = append(rows,
				[]string{"GPU Utilization", fmt.Sprintf("%.0f%%", m.GPU.UtilizationRatio*100)},
				[]string{"GPU Memory", fmt.Sprintf("%s / %s",
					styles.FormatBytes(m.GPU.MemoryUsedBytes),
					styles.FormatBytes(m.GPU.MemoryTotalBytes))},
				[]string{"GPU Temperature", fmt.Sprintf("%.0fC", m.GPU.TemperatureCelsius)},
				[]string{"GPU Power", fmt.Sprintf("%.1fW", m.GPU.PowerWatts)},
			)
		}

		fmt.Fprintln(cmd.OutOrStdout(), styles.NewTable().Rows(rows...).Render())

		return nil
	},
}

func init() {
	metricsCmd.Flags().Bool("gpu", false, "Include GPU metrics")
	InfoCmd.AddCommand(metricsCmd)
}

func formatMemory(used, total uint64) string {
	if total == 0 {
		return styles.FormatBytes(used) + " (no limit)"
	}

	return fmt.Sprintf("%s / %s (%s)",
		styles.FormatBytes(used),
		styles.FormatBytes(total),
		styles.FormatPercent(used, total))
}
