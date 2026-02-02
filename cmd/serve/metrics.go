package serve

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/env"
	internalIO "github.com/kloudkit/ws-cli/internals/io"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

const metricsNamespace = "workspace"

var validCollectors = map[string][]string{
	"workspace":            {"workspace.info", "workspace.session", "workspace.extensions"},
	"workspace.info":       {},
	"workspace.session":    {},
	"workspace.extensions": {},
	"container":            {"container.cpu", "container.memory", "container.fs", "container.fd"},
	"container.cpu":        {},
	"container.memory":     {},
	"container.fs":         {},
	"container.fd":         {},
	"gpu":                  {},
}

var allLeafCollectors = []string{
	"container.cpu",
	"container.fd",
	"container.fs",
	"container.memory",
	"workspace.extensions",
	"workspace.info",
	"workspace.session",
}

func isCollectorEnabled(name string, collectors []string) bool {
	if len(collectors) == 0 || slices.Contains(collectors, "*") {
		return true
	}

	for _, c := range collectors {
		if c == name || strings.HasPrefix(name, c+".") || strings.HasPrefix(c, name+".") {
			return true
		}
	}

	return false
}

func expandCollectors(collectors []string) []string {
	if len(collectors) == 0 || slices.Contains(collectors, "*") {
		result := make([]string, len(allLeafCollectors))
		copy(result, allLeafCollectors)
		if internalIO.IsGPUAvailable() {
			result = append(result, "gpu")
		}
		return result
	}

	expanded := make(map[string]bool)
	for _, c := range collectors {
		subs := validCollectors[c]
		if len(subs) == 0 {
			expanded[c] = true
			continue
		}
		for _, sub := range subs {
			expanded[sub] = true
		}
	}

	result := make([]string, 0, len(expanded))
	for c := range expanded {
		result = append(result, c)
	}
	slices.Sort(result)
	return result
}

func newDesc(subsystem, name, description string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(metricsNamespace, subsystem, name),
		description,
		nil, nil,
	)
}

type workspaceCollector struct {
	info                     *prometheus.Desc
	initializedTimestamp     *prometheus.Desc
	uptimeSeconds            *prometheus.Desc
	extensionsInstalledTotal *prometheus.Desc
	infoLabels               prometheus.Labels
	initializedUnix          float64
	enabled                  []string
}

func newWorkspaceCollector(enabled []string) *workspaceCollector {
	c := &workspaceCollector{
		info: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "", "info"),
			"Workspace build information",
			[]string{"version", "vscode_version"},
			nil,
		),
		initializedTimestamp:     newDesc("session", "initialized_timestamp_seconds", "Unix timestamp when workspace was initialized"),
		uptimeSeconds:            newDesc("session", "uptime_seconds", "Seconds since workspace was initialized"),
		extensionsInstalledTotal: newDesc("", "extensions_installed_total", "Number of VS Code extensions installed"),
		infoLabels:               prometheus.Labels{"version": "", "vscode_version": ""},
		enabled:                  enabled,
	}

	if manifest, err := config.ReadManifest(); err == nil {
		c.infoLabels = prometheus.Labels{
			"version":        manifest.Version,
			"vscode_version": manifest.VSCode.Version,
		}
	}

	if initialized, err := config.GetInitializedTime(); err == nil {
		c.initializedUnix = float64(initialized.Unix())
	}

	return c
}

func (c *workspaceCollector) has(name string) bool {
	return isCollectorEnabled(name, c.enabled)
}

func (c *workspaceCollector) Describe(ch chan<- *prometheus.Desc) {
	if c.has("workspace.info") {
		ch <- c.info
	}
	if c.has("workspace.session") {
		ch <- c.initializedTimestamp
		ch <- c.uptimeSeconds
	}
	if c.has("workspace.extensions") {
		ch <- c.extensionsInstalledTotal
	}
}

func (c *workspaceCollector) Collect(ch chan<- prometheus.Metric) {
	if c.has("workspace.info") {
		ch <- prometheus.MustNewConstMetric(
			c.info, prometheus.GaugeValue, 1,
			c.infoLabels["version"], c.infoLabels["vscode_version"],
		)
	}

	if c.has("workspace.session") {
		ch <- prometheus.MustNewConstMetric(c.initializedTimestamp, prometheus.GaugeValue, c.initializedUnix)
		if uptime, err := config.GetUptime(); err == nil {
			ch <- prometheus.MustNewConstMetric(c.uptimeSeconds, prometheus.GaugeValue, uptime.Seconds())
		}
	}

	if c.has("workspace.extensions") {
		if count, err := config.GetExtensionCount(); err == nil {
			ch <- prometheus.MustNewConstMetric(c.extensionsInstalledTotal, prometheus.GaugeValue, float64(count))
		}
	}
}

type containerCollector struct {
	cpuUsageSeconds  *prometheus.Desc
	cpuUserSeconds   *prometheus.Desc
	cpuSystemSeconds *prometheus.Desc
	memUsageBytes    *prometheus.Desc
	memLimitBytes    *prometheus.Desc
	memRSSBytes      *prometheus.Desc
	fsUsageBytes     *prometheus.Desc
	fsLimitBytes     *prometheus.Desc
	fdOpen           *prometheus.Desc
	fdLimit          *prometheus.Desc
	enabled          []string
}

func newContainerCollector(enabled []string) *containerCollector {
	desc := func(name, description string) *prometheus.Desc {
		return newDesc("container", name, description)
	}
	return &containerCollector{
		cpuUsageSeconds:  desc("cpu_usage_seconds_total", "Total CPU time consumed by the container"),
		cpuUserSeconds:   desc("cpu_user_seconds_total", "CPU time consumed in user mode"),
		cpuSystemSeconds: desc("cpu_system_seconds_total", "CPU time consumed in system mode"),
		memUsageBytes:    desc("memory_usage_bytes", "Current memory usage in bytes"),
		memLimitBytes:    desc("memory_limit_bytes", "Memory limit in bytes"),
		memRSSBytes:      desc("memory_rss_bytes", "Resident set size in bytes"),
		fsUsageBytes:     desc("fs_usage_bytes", "Filesystem usage in bytes on /workspace"),
		fsLimitBytes:     desc("fs_limit_bytes", "Filesystem capacity in bytes on /workspace"),
		fdOpen:           desc("file_descriptors_open", "Number of open file descriptors"),
		fdLimit:          desc("file_descriptors_limit", "File descriptor limit"),
		enabled:          enabled,
	}
}

func (c *containerCollector) has(name string) bool {
	return isCollectorEnabled(name, c.enabled)
}

func (c *containerCollector) Describe(ch chan<- *prometheus.Desc) {
	if c.has("container.cpu") {
		ch <- c.cpuUsageSeconds
		ch <- c.cpuUserSeconds
		ch <- c.cpuSystemSeconds
	}
	if c.has("container.memory") {
		ch <- c.memUsageBytes
		ch <- c.memLimitBytes
		ch <- c.memRSSBytes
	}
	if c.has("container.fs") {
		ch <- c.fsUsageBytes
		ch <- c.fsLimitBytes
	}
	if c.has("container.fd") {
		ch <- c.fdOpen
		ch <- c.fdLimit
	}
}

func (c *containerCollector) Collect(ch chan<- prometheus.Metric) {
	if c.has("container.cpu") {
		if cpu, err := internalIO.GetCPUStats(); err == nil {
			ch <- prometheus.MustNewConstMetric(c.cpuUsageSeconds, prometheus.CounterValue, cpu.UsageSeconds)
			ch <- prometheus.MustNewConstMetric(c.cpuUserSeconds, prometheus.CounterValue, cpu.UserSeconds)
			ch <- prometheus.MustNewConstMetric(c.cpuSystemSeconds, prometheus.CounterValue, cpu.SystemSeconds)
		}
	}

	if c.has("container.memory") {
		if mem, err := internalIO.GetMemoryStats(); err == nil {
			ch <- prometheus.MustNewConstMetric(c.memUsageBytes, prometheus.GaugeValue, float64(mem.UsageBytes))
			ch <- prometheus.MustNewConstMetric(c.memLimitBytes, prometheus.GaugeValue, float64(mem.LimitBytes))
			ch <- prometheus.MustNewConstMetric(c.memRSSBytes, prometheus.GaugeValue, float64(mem.RSSBytes))
		}
	}

	if c.has("container.fs") {
		if disk, err := internalIO.GetDiskStats(); err == nil {
			ch <- prometheus.MustNewConstMetric(c.fsUsageBytes, prometheus.GaugeValue, float64(disk.UsageBytes))
			ch <- prometheus.MustNewConstMetric(c.fsLimitBytes, prometheus.GaugeValue, float64(disk.LimitBytes))
		}
	}

	if c.has("container.fd") {
		if fd, err := internalIO.GetFileDescriptorStats(); err == nil {
			ch <- prometheus.MustNewConstMetric(c.fdOpen, prometheus.GaugeValue, float64(fd.Open))
			ch <- prometheus.MustNewConstMetric(c.fdLimit, prometheus.GaugeValue, float64(fd.Limit))
		}
	}
}

type gpuCollector struct {
	utilizationRatio   *prometheus.Desc
	memoryUsedBytes    *prometheus.Desc
	memoryTotalBytes   *prometheus.Desc
	temperatureCelsius *prometheus.Desc
	powerWatts         *prometheus.Desc
}

func newGPUCollector() *gpuCollector {
	desc := func(name, description string) *prometheus.Desc {
		return newDesc("gpu", name, description)
	}
	return &gpuCollector{
		utilizationRatio:   desc("utilization_ratio", "GPU utilization ratio (0-1)"),
		memoryUsedBytes:    desc("memory_used_bytes", "GPU memory used in bytes"),
		memoryTotalBytes:   desc("memory_total_bytes", "GPU memory total in bytes"),
		temperatureCelsius: desc("temperature_celsius", "GPU temperature in Celsius"),
		powerWatts:         desc("power_watts", "GPU power consumption in watts"),
	}
}

func (c *gpuCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.utilizationRatio
	ch <- c.memoryUsedBytes
	ch <- c.memoryTotalBytes
	ch <- c.temperatureCelsius
	ch <- c.powerWatts
}

func (c *gpuCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := internalIO.GetGPUStats()
	if err != nil || !stats.Available {
		return
	}

	ch <- prometheus.MustNewConstMetric(c.utilizationRatio, prometheus.GaugeValue, stats.UtilizationRatio)
	ch <- prometheus.MustNewConstMetric(c.memoryUsedBytes, prometheus.GaugeValue, float64(stats.MemoryUsedBytes))
	ch <- prometheus.MustNewConstMetric(c.memoryTotalBytes, prometheus.GaugeValue, float64(stats.MemoryTotalBytes))
	ch <- prometheus.MustNewConstMetric(c.temperatureCelsius, prometheus.GaugeValue, stats.TemperatureCelsius)
	ch <- prometheus.MustNewConstMetric(c.powerWatts, prometheus.GaugeValue, stats.PowerWatts)
}

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Start the Prometheus metrics server",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		collectors, _ := cmd.Flags().GetStringSlice("collectors")
		out := cmd.OutOrStdout()

		var validated, invalid []string
		hasExplicit := len(collectors) > 0

		for _, c := range collectors {
			if c = strings.TrimSpace(c); c == "" {
				continue
			}
			if c == "*" {
				validated = []string{"*"}
				break
			}
			if _, ok := validCollectors[c]; ok {
				validated = append(validated, c)
			} else {
				invalid = append(invalid, c)
			}
		}

		styles.PrintTitle(out, "Metrics Server")

		for _, c := range invalid {
			styles.PrintWarning(out, fmt.Sprintf("Unknown collector '%s', skipping", c))
		}

		hasWorkspace := isCollectorEnabled("workspace", validated)
		hasContainer := isCollectorEnabled("container", validated)
		gpuRequested := slices.Contains(validated, "gpu")
		hasGPU := isCollectorEnabled("gpu", validated) && internalIO.IsGPUAvailable()

		if gpuRequested && !hasGPU {
			styles.PrintWarning(out, "GPU not available, skipping gpu collector")
			validated = slices.DeleteFunc(validated, func(c string) bool { return c == "gpu" })
		}

		if hasExplicit && !hasWorkspace && !hasContainer && !hasGPU {
			return errors.New("no collectors enabled")
		}

		registry := prometheus.NewRegistry()
		if hasWorkspace {
			registry.MustRegister(newWorkspaceCollector(validated))
		}
		if hasContainer {
			registry.MustRegister(newContainerCollector(validated))
		}
		if hasGPU {
			registry.MustRegister(newGPUCollector())
		}

		fmt.Fprintln(out, styles.Info().Render("  Collectors:"))
		for _, c := range expandCollectors(validated) {
			fmt.Fprintln(out, styles.Muted().Render("\t"+c))
		}
		fmt.Fprintln(out)

		addr := fmt.Sprintf(":%d", port)
		http.Handle("/", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

		styles.PrintSuccess(out, fmt.Sprintf("Serving metrics at http://0.0.0.0%s", addr))
		fmt.Fprintln(out, styles.Info().Render("Press Ctrl+C to stop"))

		return http.ListenAndServe(addr, nil)
	},
}

func getMetricsDefaultPort() int {
	if portStr := env.String(config.EnvMetricsPort); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			return port
		}
	}
	return config.DefaultMetricsPort
}

func getMetricsDefaultCollectors() []string {
	envCollectors := env.String(config.EnvMetricsCollectors)
	if envCollectors == "" {
		return nil
	}

	var collectors []string
	for _, c := range strings.Split(envCollectors, ",") {
		if c = strings.TrimSpace(c); c != "" {
			collectors = append(collectors, c)
		}
	}
	return collectors
}

func init() {
	metricsCmd.Flags().IntP("port", "p", getMetricsDefaultPort(), "Port to serve metrics on")
	metricsCmd.Flags().StringSlice("collectors", getMetricsDefaultCollectors(), "Comma-separated list of collectors to enable (e.g., workspace,container.cpu,gpu)")

	ServeCmd.AddCommand(metricsCmd)
}
