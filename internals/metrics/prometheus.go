package metrics

import (
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/prometheus/client_golang/prometheus"
)

func newDesc(subsystem, name, description string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, subsystem, name),
		description,
		nil, nil,
	)
}

type WorkspaceCollector struct {
	info                     *prometheus.Desc
	initializedTimestamp     *prometheus.Desc
	uptimeSeconds            *prometheus.Desc
	extensionsInstalledTotal *prometheus.Desc
	infoLabels               prometheus.Labels
	initializedUnix          float64
	enabled                  []string
}

func NewWorkspaceCollector(enabled []string) *WorkspaceCollector {
	c := &WorkspaceCollector{
		info: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "", "info"),
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

func (c *WorkspaceCollector) has(name string) bool {
	return IsCollectorEnabled(name, c.enabled)
}

func (c *WorkspaceCollector) Describe(ch chan<- *prometheus.Desc) {
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

func (c *WorkspaceCollector) Collect(ch chan<- prometheus.Metric) {
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

type ContainerCollector struct {
	cpuUsageSeconds     *prometheus.Desc
	cpuUserSeconds      *prometheus.Desc
	cpuSystemSeconds    *prometheus.Desc
	cpuThrottledPeriods *prometheus.Desc
	cpuThrottledSeconds *prometheus.Desc
	cpuPeriodsTotal     *prometheus.Desc
	memUsageBytes       *prometheus.Desc
	memLimitBytes       *prometheus.Desc
	memRSSBytes         *prometheus.Desc
	memCacheBytes       *prometheus.Desc
	memSwapBytes        *prometheus.Desc
	memSwapLimitBytes   *prometheus.Desc
	memAnonBytes        *prometheus.Desc
	memKernelBytes      *prometheus.Desc
	memSlabBytes        *prometheus.Desc
	memOOMTotal         *prometheus.Desc
	memOOMKillTotal     *prometheus.Desc
	memMaxTotal         *prometheus.Desc
	fsUsageBytes        *prometheus.Desc
	fsLimitBytes        *prometheus.Desc
	fdOpen              *prometheus.Desc
	fdLimit             *prometheus.Desc
	pidsCurrent         *prometheus.Desc
	pidsLimit           *prometheus.Desc
	enabled             []string
}

func NewContainerCollector(enabled []string) *ContainerCollector {
	desc := func(name, description string) *prometheus.Desc {
		return newDesc("container", name, description)
	}
	return &ContainerCollector{
		cpuUsageSeconds:     desc("cpu_usage_seconds_total", "Total CPU time consumed by the container"),
		cpuUserSeconds:      desc("cpu_user_seconds_total", "CPU time consumed in user mode"),
		cpuSystemSeconds:    desc("cpu_system_seconds_total", "CPU time consumed in system mode"),
		cpuThrottledPeriods: desc("cpu_throttled_periods_total", "Number of throttled CPU periods"),
		cpuThrottledSeconds: desc("cpu_throttled_seconds_total", "Total time throttled in seconds"),
		cpuPeriodsTotal:     desc("cpu_periods_total", "Total number of CPU scheduling periods"),
		memUsageBytes:       desc("memory_usage_bytes", "Current memory usage in bytes"),
		memLimitBytes:       desc("memory_limit_bytes", "Memory limit in bytes"),
		memRSSBytes:         desc("memory_rss_bytes", "Resident set size in bytes"),
		memCacheBytes:       desc("memory_cache_bytes", "Page cache memory in bytes"),
		memSwapBytes:        desc("memory_swap_bytes", "Swap usage in bytes"),
		memSwapLimitBytes:   desc("memory_swap_limit_bytes", "Swap limit in bytes"),
		memAnonBytes:        desc("memory_anon_bytes", "Anonymous memory in bytes"),
		memKernelBytes:      desc("memory_kernel_bytes", "Kernel memory in bytes"),
		memSlabBytes:        desc("memory_slab_bytes", "Slab allocator memory in bytes"),
		memOOMTotal:         desc("memory_oom_total", "Number of OOM events"),
		memOOMKillTotal:     desc("memory_oom_kill_total", "Number of OOM kill events"),
		memMaxTotal:         desc("memory_max_total", "Number of times memory limit was hit"),
		fsUsageBytes:        desc("fs_usage_bytes", "Filesystem usage in bytes on /workspace"),
		fsLimitBytes:        desc("fs_limit_bytes", "Filesystem capacity in bytes on /workspace"),
		fdOpen:              desc("file_descriptors_open", "Number of open file descriptors"),
		fdLimit:             desc("file_descriptors_limit", "File descriptor limit"),
		pidsCurrent:         desc("pids_current", "Current number of processes"),
		pidsLimit:           desc("pids_limit", "Process limit"),
		enabled:             enabled,
	}
}

func (c *ContainerCollector) has(name string) bool {
	return IsCollectorEnabled(name, c.enabled)
}

func (c *ContainerCollector) Describe(ch chan<- *prometheus.Desc) {
	if c.has("container.cpu") {
		ch <- c.cpuUsageSeconds
		ch <- c.cpuUserSeconds
		ch <- c.cpuSystemSeconds
		ch <- c.cpuThrottledPeriods
		ch <- c.cpuThrottledSeconds
		ch <- c.cpuPeriodsTotal
	}
	if c.has("container.memory") {
		ch <- c.memUsageBytes
		ch <- c.memLimitBytes
		ch <- c.memRSSBytes
		ch <- c.memCacheBytes
		ch <- c.memSwapBytes
		ch <- c.memSwapLimitBytes
		ch <- c.memAnonBytes
		ch <- c.memKernelBytes
		ch <- c.memSlabBytes
		ch <- c.memOOMTotal
		ch <- c.memOOMKillTotal
		ch <- c.memMaxTotal
	}
	if c.has("container.fs") {
		ch <- c.fsUsageBytes
		ch <- c.fsLimitBytes
	}
	if c.has("container.fd") {
		ch <- c.fdOpen
		ch <- c.fdLimit
	}
	if c.has("container.pids") {
		ch <- c.pidsCurrent
		ch <- c.pidsLimit
	}
}

func (c *ContainerCollector) Collect(ch chan<- prometheus.Metric) {
	if c.has("container.cpu") {
		if cpu, err := GetCPUStats(); err == nil {
			ch <- prometheus.MustNewConstMetric(c.cpuUsageSeconds, prometheus.CounterValue, cpu.UsageSeconds)
			ch <- prometheus.MustNewConstMetric(c.cpuUserSeconds, prometheus.CounterValue, cpu.UserSeconds)
			ch <- prometheus.MustNewConstMetric(c.cpuSystemSeconds, prometheus.CounterValue, cpu.SystemSeconds)
			ch <- prometheus.MustNewConstMetric(c.cpuThrottledPeriods, prometheus.CounterValue, float64(cpu.ThrottledPeriods))
			ch <- prometheus.MustNewConstMetric(c.cpuThrottledSeconds, prometheus.CounterValue, cpu.ThrottledSeconds)
			ch <- prometheus.MustNewConstMetric(c.cpuPeriodsTotal, prometheus.CounterValue, float64(cpu.TotalPeriods))
		}
	}

	if c.has("container.memory") {
		if mem, err := GetMemoryStats(); err == nil {
			ch <- prometheus.MustNewConstMetric(c.memUsageBytes, prometheus.GaugeValue, float64(mem.UsageBytes))
			ch <- prometheus.MustNewConstMetric(c.memLimitBytes, prometheus.GaugeValue, float64(mem.LimitBytes))
			ch <- prometheus.MustNewConstMetric(c.memRSSBytes, prometheus.GaugeValue, float64(mem.RSSBytes))
			ch <- prometheus.MustNewConstMetric(c.memCacheBytes, prometheus.GaugeValue, float64(mem.CacheBytes))
			ch <- prometheus.MustNewConstMetric(c.memSwapBytes, prometheus.GaugeValue, float64(mem.SwapBytes))
			ch <- prometheus.MustNewConstMetric(c.memSwapLimitBytes, prometheus.GaugeValue, float64(mem.SwapLimitBytes))
			ch <- prometheus.MustNewConstMetric(c.memAnonBytes, prometheus.GaugeValue, float64(mem.AnonBytes))
			ch <- prometheus.MustNewConstMetric(c.memKernelBytes, prometheus.GaugeValue, float64(mem.KernelBytes))
			ch <- prometheus.MustNewConstMetric(c.memSlabBytes, prometheus.GaugeValue, float64(mem.SlabBytes))
			ch <- prometheus.MustNewConstMetric(c.memOOMTotal, prometheus.CounterValue, float64(mem.OOMEvents))
			ch <- prometheus.MustNewConstMetric(c.memOOMKillTotal, prometheus.CounterValue, float64(mem.OOMKillEvents))
			ch <- prometheus.MustNewConstMetric(c.memMaxTotal, prometheus.CounterValue, float64(mem.MaxEvents))
		}
	}

	if c.has("container.fs") {
		if disk, err := GetDiskStats(); err == nil {
			ch <- prometheus.MustNewConstMetric(c.fsUsageBytes, prometheus.GaugeValue, float64(disk.UsageBytes))
			ch <- prometheus.MustNewConstMetric(c.fsLimitBytes, prometheus.GaugeValue, float64(disk.LimitBytes))
		}
	}

	if c.has("container.fd") {
		if fd, err := GetFileDescriptorStats(); err == nil {
			ch <- prometheus.MustNewConstMetric(c.fdOpen, prometheus.GaugeValue, float64(fd.Open))
			ch <- prometheus.MustNewConstMetric(c.fdLimit, prometheus.GaugeValue, float64(fd.Limit))
		}
	}

	if c.has("container.pids") {
		if pids, err := GetPIDStats(); err == nil {
			ch <- prometheus.MustNewConstMetric(c.pidsCurrent, prometheus.GaugeValue, float64(pids.Current))
			ch <- prometheus.MustNewConstMetric(c.pidsLimit, prometheus.GaugeValue, float64(pids.Limit))
		}
	}
}

type GPUCollector struct {
	utilizationRatio   *prometheus.Desc
	memoryUsedBytes    *prometheus.Desc
	memoryTotalBytes   *prometheus.Desc
	temperatureCelsius *prometheus.Desc
	powerWatts         *prometheus.Desc
}

func NewGPUCollector() *GPUCollector {
	desc := func(name, description string) *prometheus.Desc {
		return newDesc("gpu", name, description)
	}
	return &GPUCollector{
		utilizationRatio:   desc("utilization_ratio", "GPU utilization ratio (0-1)"),
		memoryUsedBytes:    desc("memory_used_bytes", "GPU memory used in bytes"),
		memoryTotalBytes:   desc("memory_total_bytes", "GPU memory total in bytes"),
		temperatureCelsius: desc("temperature_celsius", "GPU temperature in Celsius"),
		powerWatts:         desc("power_watts", "GPU power consumption in watts"),
	}
}

func (c *GPUCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.utilizationRatio
	ch <- c.memoryUsedBytes
	ch <- c.memoryTotalBytes
	ch <- c.temperatureCelsius
	ch <- c.powerWatts
}

func (c *GPUCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := GetGPUStats()
	if err != nil || !stats.Available {
		return
	}

	ch <- prometheus.MustNewConstMetric(c.utilizationRatio, prometheus.GaugeValue, stats.UtilizationRatio)
	ch <- prometheus.MustNewConstMetric(c.memoryUsedBytes, prometheus.GaugeValue, float64(stats.MemoryUsedBytes))
	ch <- prometheus.MustNewConstMetric(c.memoryTotalBytes, prometheus.GaugeValue, float64(stats.MemoryTotalBytes))
	ch <- prometheus.MustNewConstMetric(c.temperatureCelsius, prometheus.GaugeValue, stats.TemperatureCelsius)
	ch <- prometheus.MustNewConstMetric(c.powerWatts, prometheus.GaugeValue, stats.PowerWatts)
}

type PressureCollector struct {
	cpuWaitingSeconds    *prometheus.Desc
	cpuStalledSeconds    *prometheus.Desc
	memoryWaitingSeconds *prometheus.Desc
	memoryStalledSeconds *prometheus.Desc
	ioWaitingSeconds     *prometheus.Desc
	ioStalledSeconds     *prometheus.Desc
	enabled              []string
}

func NewPressureCollector(enabled []string) *PressureCollector {
	desc := func(name, description string) *prometheus.Desc {
		return newDesc("pressure", name, description)
	}
	return &PressureCollector{
		cpuWaitingSeconds:    desc("cpu_waiting_seconds_total", "Total time tasks waited for CPU"),
		cpuStalledSeconds:    desc("cpu_stalled_seconds_total", "Total time all tasks were stalled on CPU"),
		memoryWaitingSeconds: desc("memory_waiting_seconds_total", "Total time tasks waited for memory"),
		memoryStalledSeconds: desc("memory_stalled_seconds_total", "Total time all tasks were stalled on memory"),
		ioWaitingSeconds:     desc("io_waiting_seconds_total", "Total time tasks waited for I/O"),
		ioStalledSeconds:     desc("io_stalled_seconds_total", "Total time all tasks were stalled on I/O"),
		enabled:              enabled,
	}
}

func (c *PressureCollector) has(name string) bool {
	return IsCollectorEnabled(name, c.enabled)
}

func (c *PressureCollector) Describe(ch chan<- *prometheus.Desc) {
	if c.has("pressure.cpu") {
		ch <- c.cpuWaitingSeconds
		ch <- c.cpuStalledSeconds
	}
	if c.has("pressure.memory") {
		ch <- c.memoryWaitingSeconds
		ch <- c.memoryStalledSeconds
	}
	if c.has("pressure.io") {
		ch <- c.ioWaitingSeconds
		ch <- c.ioStalledSeconds
	}
}

func (c *PressureCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := GetPressureStats()
	if err != nil {
		return
	}

	if c.has("pressure.cpu") {
		ch <- prometheus.MustNewConstMetric(c.cpuWaitingSeconds, prometheus.CounterValue, stats.CPUWaitingSeconds)
		ch <- prometheus.MustNewConstMetric(c.cpuStalledSeconds, prometheus.CounterValue, stats.CPUStalledSeconds)
	}
	if c.has("pressure.memory") {
		ch <- prometheus.MustNewConstMetric(c.memoryWaitingSeconds, prometheus.CounterValue, stats.MemoryWaitingSeconds)
		ch <- prometheus.MustNewConstMetric(c.memoryStalledSeconds, prometheus.CounterValue, stats.MemoryStalledSeconds)
	}
	if c.has("pressure.io") {
		ch <- prometheus.MustNewConstMetric(c.ioWaitingSeconds, prometheus.CounterValue, stats.IOWaitingSeconds)
		ch <- prometheus.MustNewConstMetric(c.ioStalledSeconds, prometheus.CounterValue, stats.IOStalledSeconds)
	}
}

type NetworkCollector struct {
	receiveBytesTotal    *prometheus.Desc
	transmitBytesTotal   *prometheus.Desc
	receivePacketsTotal  *prometheus.Desc
	transmitPacketsTotal *prometheus.Desc
	receiveErrorsTotal   *prometheus.Desc
	transmitErrorsTotal  *prometheus.Desc
}

func NewNetworkCollector() *NetworkCollector {
	desc := func(name, description string) *prometheus.Desc {
		return newDesc("network", name, description)
	}
	return &NetworkCollector{
		receiveBytesTotal:    desc("receive_bytes_total", "Total bytes received"),
		transmitBytesTotal:   desc("transmit_bytes_total", "Total bytes transmitted"),
		receivePacketsTotal:  desc("receive_packets_total", "Total packets received"),
		transmitPacketsTotal: desc("transmit_packets_total", "Total packets transmitted"),
		receiveErrorsTotal:   desc("receive_errors_total", "Total receive errors"),
		transmitErrorsTotal:  desc("transmit_errors_total", "Total transmit errors"),
	}
}

func (c *NetworkCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.receiveBytesTotal
	ch <- c.transmitBytesTotal
	ch <- c.receivePacketsTotal
	ch <- c.transmitPacketsTotal
	ch <- c.receiveErrorsTotal
	ch <- c.transmitErrorsTotal
}

func (c *NetworkCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := GetNetworkStats()
	if err != nil {
		return
	}

	ch <- prometheus.MustNewConstMetric(c.receiveBytesTotal, prometheus.CounterValue, float64(stats.ReceiveBytesTotal))
	ch <- prometheus.MustNewConstMetric(c.transmitBytesTotal, prometheus.CounterValue, float64(stats.TransmitBytesTotal))
	ch <- prometheus.MustNewConstMetric(c.receivePacketsTotal, prometheus.CounterValue, float64(stats.ReceivePacketsTotal))
	ch <- prometheus.MustNewConstMetric(c.transmitPacketsTotal, prometheus.CounterValue, float64(stats.TransmitPacketsTotal))
	ch <- prometheus.MustNewConstMetric(c.receiveErrorsTotal, prometheus.CounterValue, float64(stats.ReceiveErrorsTotal))
	ch <- prometheus.MustNewConstMetric(c.transmitErrorsTotal, prometheus.CounterValue, float64(stats.TransmitErrorsTotal))
}

type IOCollector struct {
	readBytesTotal  *prometheus.Desc
	writeBytesTotal *prometheus.Desc
	readOpsTotal    *prometheus.Desc
	writeOpsTotal   *prometheus.Desc
}

func NewIOCollector() *IOCollector {
	desc := func(name, description string) *prometheus.Desc {
		return newDesc("io", name, description)
	}
	return &IOCollector{
		readBytesTotal:  desc("read_bytes_total", "Total bytes read from disk"),
		writeBytesTotal: desc("write_bytes_total", "Total bytes written to disk"),
		readOpsTotal:    desc("read_ops_total", "Total disk read operations"),
		writeOpsTotal:   desc("write_ops_total", "Total disk write operations"),
	}
}

func (c *IOCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.readBytesTotal
	ch <- c.writeBytesTotal
	ch <- c.readOpsTotal
	ch <- c.writeOpsTotal
}

func (c *IOCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := GetIOStats()
	if err != nil {
		return
	}

	ch <- prometheus.MustNewConstMetric(c.readBytesTotal, prometheus.CounterValue, float64(stats.ReadBytesTotal))
	ch <- prometheus.MustNewConstMetric(c.writeBytesTotal, prometheus.CounterValue, float64(stats.WriteBytesTotal))
	ch <- prometheus.MustNewConstMetric(c.readOpsTotal, prometheus.CounterValue, float64(stats.ReadOpsTotal))
	ch <- prometheus.MustNewConstMetric(c.writeOpsTotal, prometheus.CounterValue, float64(stats.WriteOpsTotal))
}

type SocketsCollector struct {
	tcpEstablished *prometheus.Desc
	tcpListen      *prometheus.Desc
	udp            *prometheus.Desc
}

func NewSocketsCollector() *SocketsCollector {
	desc := func(name, description string) *prometheus.Desc {
		return newDesc("sockets", name, description)
	}
	return &SocketsCollector{
		tcpEstablished: desc("tcp_established", "Number of established TCP connections"),
		tcpListen:      desc("tcp_listen", "Number of listening TCP sockets"),
		udp:            desc("udp", "Number of UDP sockets"),
	}
}

func (c *SocketsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.tcpEstablished
	ch <- c.tcpListen
	ch <- c.udp
}

func (c *SocketsCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := GetSocketStats()
	if err != nil {
		return
	}

	ch <- prometheus.MustNewConstMetric(c.tcpEstablished, prometheus.GaugeValue, float64(stats.TCPEstablished))
	ch <- prometheus.MustNewConstMetric(c.tcpListen, prometheus.GaugeValue, float64(stats.TCPListen))
	ch <- prometheus.MustNewConstMetric(c.udp, prometheus.GaugeValue, float64(stats.UDP))
}
