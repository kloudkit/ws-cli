package info

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/kloudkit/ws-cli/internals/styles"
)

type resources struct {
	cpuUsage    float64
	memoryTotal uint64
	memoryUsed  uint64
	diskTotal   uint64
	diskUsed    uint64
}

func getCPUTimes() (idle, total uint64, err error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, 0, fmt.Errorf("failed to read /proc/stat")
	}

	line := scanner.Text()
	if !strings.HasPrefix(line, "cpu ") {
		return 0, 0, fmt.Errorf("invalid /proc/stat format")
	}

	fields := strings.Fields(line)
	if len(fields) < 5 {
		return 0, 0, fmt.Errorf("invalid /proc/stat format")
	}

	var times []uint64
	for i := 1; i < len(fields); i++ {
		val, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		times = append(times, val)
		total += val
	}

	if len(times) >= 4 {
		idle = times[3]
	}

	return idle, total, nil
}

func getCPUUsage() (float64, error) {
	idle1, total1, err := getCPUTimes()
	if err != nil {
		return 0, err
	}

	time.Sleep(100 * time.Millisecond)

	idle2, total2, err := getCPUTimes()
	if err != nil {
		return 0, err
	}

	idleDelta := idle2 - idle1
	totalDelta := total2 - total1

	if totalDelta == 0 {
		return 0, nil
	}

	usage := 100.0 * (1.0 - float64(idleDelta)/float64(totalDelta))
	return usage, nil
}

func getResources() (*resources, error) {
	cpuUsage, err := getCPUUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	var stat syscall.Sysinfo_t
	if err := syscall.Sysinfo(&stat); err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/"
	}

	var diskStat syscall.Statfs_t
	if err := syscall.Statfs(cwd, &diskStat); err != nil {
		return nil, fmt.Errorf("failed to get disk info: %w", err)
	}

	memTotal := stat.Totalram * uint64(stat.Unit)

	diskTotal := diskStat.Blocks * uint64(diskStat.Bsize)

	return &resources{
		cpuUsage:    cpuUsage,
		memoryTotal: memTotal,
		memoryUsed:  memTotal - stat.Freeram*uint64(stat.Unit),
		diskTotal:   diskTotal,
		diskUsed:    diskTotal - diskStat.Bavail*uint64(diskStat.Bsize),
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
			[]string{"CPU", styles.Value().Render(fmt.Sprintf("%.1f%%", res.cpuUsage))},
			[]string{"Memory", styles.Value().Render(fmt.Sprintf("%s / %s (%s)",
				styles.FormatBytes(res.memoryUsed),
				styles.FormatBytes(res.memoryTotal),
				styles.FormatPercent(res.memoryUsed, res.memoryTotal)))},
			[]string{"Disk", styles.Value().Render(fmt.Sprintf("%s / %s (%s)",
				styles.FormatBytes(res.diskUsed),
				styles.FormatBytes(res.diskTotal),
				styles.FormatPercent(res.diskUsed, res.diskTotal)))},
		)

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", t.Render())

		return nil
	},
}

func init() {
	InfoCmd.AddCommand(resourcesCmd)
}
