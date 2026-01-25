package serve

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/kloudkit/ws-cli/internals/config"
	internalIO "github.com/kloudkit/ws-cli/internals/io"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

const metricsNamespace = "workspace"

type workspaceCollector struct {
	info                     *prometheus.Desc
	initializedTimestamp     *prometheus.Desc
	uptimeSeconds            *prometheus.Desc
	extensionsInstalledTotal *prometheus.Desc
	infoLabels               prometheus.Labels
	initializedUnix          float64
}

func newWorkspaceCollector() *workspaceCollector {
	c := &workspaceCollector{
		info: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "", "info"),
			"Workspace build information",
			[]string{"version", "vscode_version"},
			nil,
		),
		initializedTimestamp: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "session", "initialized_timestamp_seconds"),
			"Unix timestamp when workspace was initialized",
			nil, nil,
		),
		uptimeSeconds: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "session", "uptime_seconds"),
			"Seconds since workspace was initialized",
			nil, nil,
		),
		extensionsInstalledTotal: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "", "extensions_installed_total"),
			"Number of VS Code extensions installed",
			nil, nil,
		),
		infoLabels: prometheus.Labels{"version": "", "vscode_version": ""},
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

func (c *workspaceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.info
	ch <- c.initializedTimestamp
	ch <- c.uptimeSeconds
	ch <- c.extensionsInstalledTotal
}

func (c *workspaceCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		c.info, prometheus.GaugeValue, 1,
		c.infoLabels["version"], c.infoLabels["vscode_version"],
	)

	ch <- prometheus.MustNewConstMetric(
		c.initializedTimestamp, prometheus.GaugeValue, c.initializedUnix,
	)

	if uptime, err := config.GetUptime(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.uptimeSeconds, prometheus.GaugeValue, uptime.Seconds())
	}

	if count, err := config.GetExtensionCount(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.extensionsInstalledTotal, prometheus.GaugeValue, float64(count))
	}
}

type containerCollector struct {
	cpuUsageSeconds  *prometheus.Desc
	cpuUserSeconds   *prometheus.Desc
	cpuSystemSeconds *prometheus.Desc
	memoryUsageBytes *prometheus.Desc
	memoryLimitBytes *prometheus.Desc
	memoryRSSBytes   *prometheus.Desc
	fsUsageBytes     *prometheus.Desc
	fsLimitBytes     *prometheus.Desc
	fdOpen           *prometheus.Desc
	fdLimit          *prometheus.Desc
}

func newContainerCollector() *containerCollector {
	return &containerCollector{
		cpuUsageSeconds: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "container", "cpu_usage_seconds_total"),
			"Total CPU time consumed by the container",
			nil, nil,
		),
		cpuUserSeconds: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "container", "cpu_user_seconds_total"),
			"CPU time consumed in user mode",
			nil, nil,
		),
		cpuSystemSeconds: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "container", "cpu_system_seconds_total"),
			"CPU time consumed in system mode",
			nil, nil,
		),
		memoryUsageBytes: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "container", "memory_usage_bytes"),
			"Current memory usage in bytes",
			nil, nil,
		),
		memoryLimitBytes: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "container", "memory_limit_bytes"),
			"Memory limit in bytes",
			nil, nil,
		),
		memoryRSSBytes: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "container", "memory_rss_bytes"),
			"Resident set size in bytes",
			nil, nil,
		),
		fsUsageBytes: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "container", "fs_usage_bytes"),
			"Filesystem usage in bytes on /workspace",
			nil, nil,
		),
		fsLimitBytes: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "container", "fs_limit_bytes"),
			"Filesystem capacity in bytes on /workspace",
			nil, nil,
		),
		fdOpen: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "container", "file_descriptors_open"),
			"Number of open file descriptors",
			nil, nil,
		),
		fdLimit: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "container", "file_descriptors_limit"),
			"File descriptor limit",
			nil, nil,
		),
	}
}

func (c *containerCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.cpuUsageSeconds
	ch <- c.cpuUserSeconds
	ch <- c.cpuSystemSeconds
	ch <- c.memoryUsageBytes
	ch <- c.memoryLimitBytes
	ch <- c.memoryRSSBytes
	ch <- c.fsUsageBytes
	ch <- c.fsLimitBytes
	ch <- c.fdOpen
	ch <- c.fdLimit
}

func (c *containerCollector) Collect(ch chan<- prometheus.Metric) {
	if cpu, err := internalIO.GetCPUStats(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.cpuUsageSeconds, prometheus.CounterValue, cpu.UsageSeconds)
		ch <- prometheus.MustNewConstMetric(c.cpuUserSeconds, prometheus.CounterValue, cpu.UserSeconds)
		ch <- prometheus.MustNewConstMetric(c.cpuSystemSeconds, prometheus.CounterValue, cpu.SystemSeconds)
	}

	if mem, err := internalIO.GetMemoryStats(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.memoryUsageBytes, prometheus.GaugeValue, float64(mem.UsageBytes))
		ch <- prometheus.MustNewConstMetric(c.memoryLimitBytes, prometheus.GaugeValue, float64(mem.LimitBytes))
		ch <- prometheus.MustNewConstMetric(c.memoryRSSBytes, prometheus.GaugeValue, float64(mem.RSSBytes))
	}

	if disk, err := internalIO.GetDiskStats(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.fsUsageBytes, prometheus.GaugeValue, float64(disk.UsageBytes))
		ch <- prometheus.MustNewConstMetric(c.fsLimitBytes, prometheus.GaugeValue, float64(disk.LimitBytes))
	}

	if fd, err := internalIO.GetFileDescriptorStats(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.fdOpen, prometheus.GaugeValue, float64(fd.Open))
		ch <- prometheus.MustNewConstMetric(c.fdLimit, prometheus.GaugeValue, float64(fd.Limit))
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
	return &gpuCollector{
		utilizationRatio: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "gpu", "utilization_ratio"),
			"GPU utilization ratio (0-1)",
			nil, nil,
		),
		memoryUsedBytes: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "gpu", "memory_used_bytes"),
			"GPU memory used in bytes",
			nil, nil,
		),
		memoryTotalBytes: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "gpu", "memory_total_bytes"),
			"GPU memory total in bytes",
			nil, nil,
		),
		temperatureCelsius: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "gpu", "temperature_celsius"),
			"GPU temperature in Celsius",
			nil, nil,
		),
		powerWatts: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "gpu", "power_watts"),
			"GPU power consumption in watts",
			nil, nil,
		),
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
		gpu, _ := cmd.Flags().GetBool("gpu")

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Title().Render("Metrics Server"))

		registry := prometheus.NewRegistry()
		registry.MustRegister(newWorkspaceCollector())
		registry.MustRegister(newContainerCollector())

		if gpu && internalIO.IsGPUAvailable() {
			registry.MustRegister(newGPUCollector())
			fmt.Fprintln(cmd.OutOrStdout(), styles.Info().Render("GPU metrics enabled"))
		}

		addr := fmt.Sprintf(":%d", port)

		http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

		fmt.Fprintln(cmd.OutOrStdout(), styles.Success().Render(fmt.Sprintf("Serving metrics at http://0.0.0.0%s/metrics", addr)))
		fmt.Fprintln(cmd.OutOrStdout(), styles.Info().Render("Press Ctrl+C to stop"))

		return http.ListenAndServe(addr, nil)
	},
}

func getMetricsDefaultPort() int {
	envPort := os.Getenv(config.EnvMetricsPort)

	if envPort == "" {
		return config.DefaultMetricsPort
	}

	port, err := strconv.Atoi(envPort)
	if err != nil {
		return config.DefaultMetricsPort
	}

	return port
}

func getMetricsDefaultGPU() bool {
	envGPU := os.Getenv(config.EnvMetricsGPU)

	return envGPU == "true" || envGPU == "1"
}

func init() {
	metricsCmd.Flags().IntP("port", "p", getMetricsDefaultPort(), "Port to serve metrics on")
	metricsCmd.Flags().Bool("gpu", getMetricsDefaultGPU(), "Enable GPU metrics collection")

	ServeCmd.AddCommand(metricsCmd)
}
