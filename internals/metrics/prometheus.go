package metrics

import (
	"errors"
	"slices"
	"time"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	counter = prometheus.CounterValue
	gauge   = prometheus.GaugeValue
)

type metric[T any] struct {
	leaf      string
	desc      *prometheus.Desc
	valueType prometheus.ValueType
	val       func(*T) float64
}

// group emits its metrics from a single fetch per scrape, and nothing when the fetch fails.
type group[T any] struct {
	fetch   func() (*T, error)
	metrics []metric[T]
}

type collectorGroup interface {
	leaves() []string
	only(enabled func(leaf string) bool) prometheus.Collector
}

func (g *group[T]) leaves() []string {
	leaves := make([]string, len(g.metrics))
	for i, m := range g.metrics {
		leaves[i] = m.leaf
	}
	return leaves
}

func (g *group[T]) only(enabled func(string) bool) prometheus.Collector {
	metrics := slices.DeleteFunc(slices.Clone(g.metrics), func(m metric[T]) bool { return !enabled(m.leaf) })
	if len(metrics) == 0 {
		return nil
	}
	return &group[T]{g.fetch, metrics}
}

func (g *group[T]) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range g.metrics {
		ch <- m.desc
	}
}

func (g *group[T]) Collect(ch chan<- prometheus.Metric) {
	stats, err := g.fetch()
	if err != nil {
		return
	}
	for _, m := range g.metrics {
		ch <- prometheus.MustNewConstMetric(m.desc, m.valueType, m.val(stats))
	}
}

func newDesc(subsystem, name, description string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(Namespace, subsystem, name), description, nil, nil)
}

func static[T any](v T) func() (*T, error) {
	return func() (*T, error) { return &v, nil }
}

func pointer[T any](fetch func() (T, error)) func() (*T, error) {
	return func() (*T, error) {
		v, err := fetch()
		return &v, err
	}
}

func availableGPUStats() (*GPUStats, error) {
	if stats := GetGPUStats(); stats.Available {
		return stats, nil
	}
	return nil, errors.New("GPU not available")
}

func collectorGroups() []collectorGroup {
	vscodeVersion, _ := config.VSCodeVersion()
	info := prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, "", "info"),
		"Workspace build information",
		nil,
		prometheus.Labels{"version": config.WorkspaceVersion(), "vscode_version": vscodeVersion},
	)

	var initializedUnix float64
	if initialized, err := config.GetInitializedTime(); err == nil {
		initializedUnix = float64(initialized.Unix())
	}

	one := func(*struct{}) float64 { return 1 }
	self := func(v *float64) float64 { return *v }
	container := func(name, description string) *prometheus.Desc { return newDesc("container", name, description) }

	return []collectorGroup{
		&group[struct{}]{static(struct{}{}), []metric[struct{}]{
			{"workspace.info", info, gauge, one},
		}},
		&group[float64]{static(initializedUnix), []metric[float64]{
			{"workspace.session", newDesc("session", "initialized_timestamp_seconds", "Unix timestamp when workspace was initialized"), gauge, self},
		}},
		&group[time.Duration]{pointer(config.GetUptime), []metric[time.Duration]{
			{"workspace.session", newDesc("session", "uptime_seconds", "Seconds since workspace was initialized"), gauge, func(d *time.Duration) float64 { return d.Seconds() }},
		}},
		&group[int]{pointer(config.GetExtensionCount), []metric[int]{
			{"workspace.extensions", newDesc("", "extensions_installed_total", "Number of VS Code extensions installed"), gauge, func(n *int) float64 { return float64(*n) }},
		}},
		&group[CPUStats]{GetCPUStats, []metric[CPUStats]{
			{"container.cpu", container("cpu_usage_seconds_total", "Total CPU time consumed by the container"), counter, func(s *CPUStats) float64 { return s.UsageSeconds }},
			{"container.cpu", container("cpu_user_seconds_total", "CPU time consumed in user mode"), counter, func(s *CPUStats) float64 { return s.UserSeconds }},
			{"container.cpu", container("cpu_system_seconds_total", "CPU time consumed in system mode"), counter, func(s *CPUStats) float64 { return s.SystemSeconds }},
			{"container.cpu", container("cpu_throttled_periods_total", "Number of throttled CPU periods"), counter, func(s *CPUStats) float64 { return float64(s.ThrottledPeriods) }},
			{"container.cpu", container("cpu_throttled_seconds_total", "Total time throttled in seconds"), counter, func(s *CPUStats) float64 { return s.ThrottledSeconds }},
			{"container.cpu", container("cpu_periods_total", "Total number of CPU scheduling periods"), counter, func(s *CPUStats) float64 { return float64(s.TotalPeriods) }},
		}},
		&group[MemoryStats]{GetMemoryStats, []metric[MemoryStats]{
			{"container.memory", container("memory_usage_bytes", "Current memory usage in bytes"), gauge, func(s *MemoryStats) float64 { return float64(s.UsageBytes) }},
			{"container.memory", container("memory_limit_bytes", "Memory limit in bytes"), gauge, func(s *MemoryStats) float64 { return float64(s.LimitBytes) }},
			{"container.memory", container("memory_rss_bytes", "Resident set size in bytes"), gauge, func(s *MemoryStats) float64 { return float64(s.RSSBytes) }},
			{"container.memory", container("memory_cache_bytes", "Page cache memory in bytes"), gauge, func(s *MemoryStats) float64 { return float64(s.CacheBytes) }},
			{"container.memory", container("memory_swap_bytes", "Swap usage in bytes"), gauge, func(s *MemoryStats) float64 { return float64(s.SwapBytes) }},
			{"container.memory", container("memory_swap_limit_bytes", "Swap limit in bytes"), gauge, func(s *MemoryStats) float64 { return float64(s.SwapLimitBytes) }},
			{"container.memory", container("memory_anon_bytes", "Anonymous memory in bytes"), gauge, func(s *MemoryStats) float64 { return float64(s.AnonBytes) }},
			{"container.memory", container("memory_kernel_bytes", "Kernel memory in bytes"), gauge, func(s *MemoryStats) float64 { return float64(s.KernelBytes) }},
			{"container.memory", container("memory_slab_bytes", "Slab allocator memory in bytes"), gauge, func(s *MemoryStats) float64 { return float64(s.SlabBytes) }},
			{"container.memory", container("memory_oom_total", "Number of OOM events"), counter, func(s *MemoryStats) float64 { return float64(s.OOMEvents) }},
			{"container.memory", container("memory_oom_kill_total", "Number of OOM kill events"), counter, func(s *MemoryStats) float64 { return float64(s.OOMKillEvents) }},
			{"container.memory", container("memory_max_total", "Number of times memory limit was hit"), counter, func(s *MemoryStats) float64 { return float64(s.MaxEvents) }},
		}},
		&group[DiskStats]{GetDiskStats, []metric[DiskStats]{
			{"container.fs", container("fs_usage_bytes", "Filesystem usage in bytes on /workspace"), gauge, func(s *DiskStats) float64 { return float64(s.UsageBytes) }},
			{"container.fs", container("fs_limit_bytes", "Filesystem capacity in bytes on /workspace"), gauge, func(s *DiskStats) float64 { return float64(s.LimitBytes) }},
		}},
		&group[FileDescriptorStats]{GetFileDescriptorStats, []metric[FileDescriptorStats]{
			{"container.fd", container("file_descriptors_open", "Number of open file descriptors"), gauge, func(s *FileDescriptorStats) float64 { return float64(s.Open) }},
			{"container.fd", container("file_descriptors_limit", "File descriptor limit"), gauge, func(s *FileDescriptorStats) float64 { return float64(s.Limit) }},
		}},
		&group[PIDStats]{GetPIDStats, []metric[PIDStats]{
			{"container.pids", container("pids_current", "Current number of processes"), gauge, func(s *PIDStats) float64 { return float64(s.Current) }},
			{"container.pids", container("pids_limit", "Process limit"), gauge, func(s *PIDStats) float64 { return float64(s.Limit) }},
		}},
		&group[PressureStats]{GetPressureStats, []metric[PressureStats]{
			{"pressure.cpu", newDesc("pressure", "cpu_waiting_seconds_total", "Total time tasks waited for CPU"), counter, func(s *PressureStats) float64 { return s.CPUWaitingSeconds }},
			{"pressure.cpu", newDesc("pressure", "cpu_stalled_seconds_total", "Total time all tasks were stalled on CPU"), counter, func(s *PressureStats) float64 { return s.CPUStalledSeconds }},
			{"pressure.memory", newDesc("pressure", "memory_waiting_seconds_total", "Total time tasks waited for memory"), counter, func(s *PressureStats) float64 { return s.MemoryWaitingSeconds }},
			{"pressure.memory", newDesc("pressure", "memory_stalled_seconds_total", "Total time all tasks were stalled on memory"), counter, func(s *PressureStats) float64 { return s.MemoryStalledSeconds }},
			{"pressure.io", newDesc("pressure", "io_waiting_seconds_total", "Total time tasks waited for I/O"), counter, func(s *PressureStats) float64 { return s.IOWaitingSeconds }},
			{"pressure.io", newDesc("pressure", "io_stalled_seconds_total", "Total time all tasks were stalled on I/O"), counter, func(s *PressureStats) float64 { return s.IOStalledSeconds }},
		}},
		&group[NetworkStats]{GetNetworkStats, []metric[NetworkStats]{
			{"network", newDesc("network", "receive_bytes_total", "Total bytes received"), counter, func(s *NetworkStats) float64 { return float64(s.ReceiveBytesTotal) }},
			{"network", newDesc("network", "transmit_bytes_total", "Total bytes transmitted"), counter, func(s *NetworkStats) float64 { return float64(s.TransmitBytesTotal) }},
			{"network", newDesc("network", "receive_packets_total", "Total packets received"), counter, func(s *NetworkStats) float64 { return float64(s.ReceivePacketsTotal) }},
			{"network", newDesc("network", "transmit_packets_total", "Total packets transmitted"), counter, func(s *NetworkStats) float64 { return float64(s.TransmitPacketsTotal) }},
			{"network", newDesc("network", "receive_errors_total", "Total receive errors"), counter, func(s *NetworkStats) float64 { return float64(s.ReceiveErrorsTotal) }},
			{"network", newDesc("network", "transmit_errors_total", "Total transmit errors"), counter, func(s *NetworkStats) float64 { return float64(s.TransmitErrorsTotal) }},
		}},
		&group[IOStats]{GetIOStats, []metric[IOStats]{
			{"io", newDesc("io", "read_bytes_total", "Total bytes read from disk"), counter, func(s *IOStats) float64 { return float64(s.ReadBytesTotal) }},
			{"io", newDesc("io", "write_bytes_total", "Total bytes written to disk"), counter, func(s *IOStats) float64 { return float64(s.WriteBytesTotal) }},
			{"io", newDesc("io", "read_ops_total", "Total disk read operations"), counter, func(s *IOStats) float64 { return float64(s.ReadOpsTotal) }},
			{"io", newDesc("io", "write_ops_total", "Total disk write operations"), counter, func(s *IOStats) float64 { return float64(s.WriteOpsTotal) }},
		}},
		&group[SocketStats]{GetSocketStats, []metric[SocketStats]{
			{"sockets", newDesc("sockets", "tcp_established", "Number of established TCP connections"), gauge, func(s *SocketStats) float64 { return float64(s.TCPEstablished) }},
			{"sockets", newDesc("sockets", "tcp_listen", "Number of listening TCP sockets"), gauge, func(s *SocketStats) float64 { return float64(s.TCPListen) }},
			{"sockets", newDesc("sockets", "udp", "Number of UDP sockets"), gauge, func(s *SocketStats) float64 { return float64(s.UDP) }},
		}},
		&group[GPUStats]{availableGPUStats, []metric[GPUStats]{
			{"gpu", newDesc("gpu", "utilization_ratio", "GPU utilization ratio (0-1)"), gauge, func(s *GPUStats) float64 { return s.UtilizationRatio }},
			{"gpu", newDesc("gpu", "memory_used_bytes", "GPU memory used in bytes"), gauge, func(s *GPUStats) float64 { return float64(s.MemoryUsedBytes) }},
			{"gpu", newDesc("gpu", "memory_total_bytes", "GPU memory total in bytes"), gauge, func(s *GPUStats) float64 { return float64(s.MemoryTotalBytes) }},
			{"gpu", newDesc("gpu", "temperature_celsius", "GPU temperature in Celsius"), gauge, func(s *GPUStats) float64 { return s.TemperatureCelsius }},
			{"gpu", newDesc("gpu", "power_watts", "GPU power consumption in watts"), gauge, func(s *GPUStats) float64 { return s.PowerWatts }},
		}},
	}
}
