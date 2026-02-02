package info

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kloudkit/ws-cli/internals/metrics"
	"github.com/kloudkit/ws-cli/internals/styles"
)

type workspaceMetrics struct {
	cpuUsage    float64
	cpuSeconds  float64
	memoryTotal uint64
	memoryUsed  uint64
	memoryRSS   uint64
	diskTotal   uint64
	diskUsed    uint64
	fdOpen      uint64
	fdLimit     uint64
	gpu         *metrics.GPUStats
}

func formatCPUTime(seconds float64) string {
	switch {
	case seconds < 60:
		return fmt.Sprintf("%.1fs", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%.1fm", seconds/60)
	default:
		return fmt.Sprintf("%.1fh", seconds/3600)
	}
}

func getMetrics(includeGPU bool) (*workspaceMetrics, error) {
	cpuUsage, _ := metrics.GetCPUUsagePercent()

	cpuStats, err := metrics.GetCPUStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU stats: %w", err)
	}

	memStats, err := metrics.GetMemoryStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory stats: %w", err)
	}

	diskStats, err := metrics.GetDiskStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk stats: %w", err)
	}

	fdStats, err := metrics.GetFileDescriptorStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get file descriptor stats: %w", err)
	}

	m := &workspaceMetrics{
		cpuUsage:    cpuUsage,
		cpuSeconds:  cpuStats.UsageSeconds,
		memoryTotal: memStats.LimitBytes,
		memoryUsed:  memStats.UsageBytes,
		memoryRSS:   memStats.RSSBytes,
		diskTotal:   diskStats.LimitBytes,
		diskUsed:    diskStats.UsageBytes,
		fdOpen:      fdStats.Open,
		fdLimit:     fdStats.Limit,
	}

	if includeGPU {
		if gpuStats, err := metrics.GetGPUStats(); err == nil {
			m.gpu = gpuStats
		}
	}

	return m, nil
}

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Display workspace metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		includeGPU, _ := cmd.Flags().GetBool("gpu")

		m, err := getMetrics(includeGPU)
		if err != nil {
			styles.PrintWarning(cmd.OutOrStdout(), "Could not read workspace metrics")
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Title().Render("Metrics"))

		rows := [][]string{
			{"CPU", fmt.Sprintf("%.1f%% (%s)", m.cpuUsage, formatCPUTime(m.cpuSeconds))},
			{"Memory", fmt.Sprintf("%s / %s (%s)",
				styles.FormatBytes(m.memoryUsed),
				styles.FormatBytes(m.memoryTotal),
				styles.FormatPercent(m.memoryUsed, m.memoryTotal))},
			{"Memory RSS", styles.FormatBytes(m.memoryRSS)},
			{"Disk", fmt.Sprintf("%s / %s (%s)",
				styles.FormatBytes(m.diskUsed),
				styles.FormatBytes(m.diskTotal),
				styles.FormatPercent(m.diskUsed, m.diskTotal))},
			{"File Descriptors", fmt.Sprintf("%d / %d", m.fdOpen, m.fdLimit)},
		}

		if m.gpu != nil && m.gpu.Available {
			rows = append(rows,
				[]string{"GPU Utilization", fmt.Sprintf("%.0f%%", m.gpu.UtilizationRatio*100)},
				[]string{"GPU Memory", fmt.Sprintf("%s / %s",
					styles.FormatBytes(m.gpu.MemoryUsedBytes),
					styles.FormatBytes(m.gpu.MemoryTotalBytes))},
				[]string{"GPU Temperature", fmt.Sprintf("%.0fC", m.gpu.TemperatureCelsius)},
				[]string{"GPU Power", fmt.Sprintf("%.1fW", m.gpu.PowerWatts)},
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
