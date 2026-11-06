package info

import (
	"fmt"
	"maps"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/kloudkit/ws-cli/cmd/cli"
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/kloudkit/ws-cli/internals/metrics"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

func Version() string {
	if build, ok := debug.ReadBuildInfo(); ok && build.Main.Version != "" {
		return build.Main.Version
	}

	return "(devel)"
}

var versionCmd = cli.New("version", "0.2.0", "Display installed workspace version",
	"Print the workspace version. --all expands to the full table: workspace, ws-cli, and VS Code.",
	func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		if !cli.Bool(cmd, "all") {
			fmt.Fprintln(out, Version())
			return nil
		}

		var rows [][]string

		if version := config.WorkspaceVersion(); version != "" {
			rows = append(rows, []string{"workspace", version})
		} else {
			styles.PrintWarning(out, "Could not determine the workspace version")
		}

		rows = append(rows, []string{"ws-cli", Version()})

		if version, err := config.VSCodeVersion(); err == nil {
			rows = append(rows, []string{"VSCode", version})
		} else {
			styles.PrintWarning(out, fmt.Sprintf("Could not read the editor version: %v", err))
		}

		styles.NewTable().Title("Versions").Rows(rows...).Print(out)

		return nil
	})

var envCmd = cli.New("env", "0.2.0", "Display effective workspace environment variables",
	"Print every WS_* variable in effect, sorted: the resolved environment the workspace booted with.",
	func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		all := env.GetAll()

		var rows [][]string
		for _, key := range slices.Sorted(maps.Keys(all)) {
			if strings.HasPrefix(key, "WS_") {
				rows = append(rows, []string{key, all[key]})
			}
		}

		if len(rows) == 0 {
			styles.PrintEmpty(out, "No WS_* variables set")
			return nil
		}

		styles.NewTable().Title("Workspace Variables", len(rows)).Rows(rows...).Print(out)

		return nil
	})

var extensionsCmd = cli.New("extensions", "0.2.0", "Display installed extensions",
	"List the installed VS Code extensions with their versions.",
	func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		extensions, err := config.GetExtensions()
		if err != nil {
			return err
		}

		if len(extensions) == 0 {
			styles.PrintEmpty(out, "No extensions installed")
			return nil
		}

		rows := make([][]string, len(extensions))
		for i, ext := range extensions {
			rows[i] = []string{ext.Name, ext.Version}
		}

		styles.NewTable("Name", "Version").Title("Extensions", len(rows)).Rows(rows...).Print(out)

		return nil
	})

var metricsCmd = cli.New("metrics", "0.2.0", "Display workspace metrics",
	"Show live resource usage: CPU, memory, disk, and file descriptors, plus GPU with --gpu.",
	func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		includeGPU := cli.Bool(cmd, "gpu")

		m, err := metrics.GetWorkspaceSummary(includeGPU)
		if err != nil {
			styles.PrintWarning(out, "Could not read workspace metrics")
			return nil
		}

		memory := styles.FormatBytes(m.MemoryUsed) + " " + styles.Aside("no limit")
		if m.MemoryTotal > 0 {
			memory = usage(m.MemoryUsed, m.MemoryTotal)
		}

		rows := [][]string{
			{"CPU", fmt.Sprintf("%.1f%% %s", m.CPUUsage, styles.Muted.Render("· "+styles.FormatCPUTime(m.CPUSeconds)+" CPU time"))},
			{"Memory", memory},
			{"Memory RSS", styles.FormatBytes(m.MemoryRSS)},
			{"Disk", usage(m.DiskUsed, m.DiskTotal)},
			{"File Descriptors", styles.FormatCount(m.FDOpen) + " open " + styles.Muted.Render("· limit "+styles.FormatCount(m.FDLimit)+" per process")},
		}

		switch {
		case !includeGPU:
		case m.GPU == nil || !m.GPU.Available:
			rows = append(rows, []string{"GPU", styles.Muted.Render("not available")})
		default:
			rows = append(rows,
				[]string{"GPU Utilization", fmt.Sprintf("%.0f%%", m.GPU.UtilizationRatio*100)},
				[]string{"GPU Memory", usage(m.GPU.MemoryUsedBytes, m.GPU.MemoryTotalBytes)},
				[]string{"GPU Temperature", fmt.Sprintf("%.0f°C", m.GPU.TemperatureCelsius)},
				[]string{"GPU Power", fmt.Sprintf("%.1f W", m.GPU.PowerWatts)},
			)
		}

		styles.NewTable().Title("Metrics").Rows(rows...).Print(out)

		return nil
	})

var uptimeCmd = cli.New("uptime", "0.2.0", "Display the workspace uptime",
	"Show when the workspace session started and how long it has been running.",
	func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		started, err := config.GetInitializedTime()
		if err != nil {
			styles.PrintWarning(out, "Could not read workspace startup time")
			return nil
		}

		running := time.Since(started)

		status := styles.Success.Render("Recently started")
		switch {
		case running.Hours() >= 36:
			status = styles.Warning.Render("Long running")
		case running.Hours() >= 1:
			status = styles.Info.Render("Active")
		}

		styles.NewTable().Title("Uptime").Rows(
			[]string{"Started at", started.Format("2006-01-02 15:04:05 MST")},
			[]string{"Running for", styles.FormatDuration(running)},
			[]string{"Status", status},
		).Print(out)

		return nil
	})

var InfoCmd = cli.Group("info", "0.2.0", "Display workspace information",
	"Report facts about the running workspace: version, effective environment, installed extensions, live resource metrics, and uptime.",
	`# Show the full version table
ws-cli info version --all

# Watch live resource usage
ws-cli info metrics`,
	envCmd, extensionsCmd, metricsCmd, uptimeCmd, versionCmd)

func usage(used, total uint64) string {
	return fmt.Sprintf("%s / %s %s", styles.FormatBytes(used), styles.FormatBytes(total), styles.Aside(styles.FormatPercent(used, total)))
}

func init() {
	versionCmd.Flags().Bool("all", false, "Show all version information")
	metricsCmd.Flags().Bool("gpu", false, "Include GPU metrics")
}
