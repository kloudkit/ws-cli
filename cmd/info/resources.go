package info

import (
	"fmt"

	"github.com/spf13/cobra"

	internalIO "github.com/kloudkit/ws-cli/internals/io"
	"github.com/kloudkit/ws-cli/internals/styles"
)

type resources struct {
	cpuUsage    float64
	cpuSeconds  float64
	memoryTotal uint64
	memoryUsed  uint64
	diskTotal   uint64
	diskUsed    uint64
}

func formatCPUTime(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}

	if seconds < 3600 {
		return fmt.Sprintf("%.1fm", seconds/60)
	}

	return fmt.Sprintf("%.1fh", seconds/3600)
}

func getResources() (*resources, error) {
	cpuUsage, _ := internalIO.GetCPUUsagePercent()

	cpuStats, err := internalIO.GetCPUStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU stats: %w", err)
	}

	memStats, err := internalIO.GetMemoryStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory stats: %w", err)
	}

	diskStats, err := internalIO.GetDiskStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk stats: %w", err)
	}

	return &resources{
		cpuUsage:    cpuUsage,
		cpuSeconds:  cpuStats.UsageSeconds,
		memoryTotal: memStats.LimitBytes,
		memoryUsed:  memStats.UsageBytes,
		diskTotal:   diskStats.LimitBytes,
		diskUsed:    diskStats.UsageBytes,
	}, nil
}

var resourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "Display system resource usage",
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := getResources()
		if err != nil {
			styles.PrintWarning(cmd.OutOrStdout(), "Could not read system resources")
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Title().Render("Resources"))

		t := styles.Table().Rows(
			[]string{"CPU", fmt.Sprintf("%.1f%% (%s)", res.cpuUsage, formatCPUTime(res.cpuSeconds))},
			[]string{"Memory", fmt.Sprintf("%s / %s (%s)",
				styles.FormatBytes(res.memoryUsed),
				styles.FormatBytes(res.memoryTotal),
				styles.FormatPercent(res.memoryUsed, res.memoryTotal))},
			[]string{"Disk", fmt.Sprintf("%s / %s (%s)",
				styles.FormatBytes(res.diskUsed),
				styles.FormatBytes(res.diskTotal),
				styles.FormatPercent(res.diskUsed, res.diskTotal))},
		)

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", t.Render())

		return nil
	},
}

func init() {
	InfoCmd.AddCommand(resourcesCmd)
}
