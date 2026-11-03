package metrics

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/kloudkit/ws-cli/internals/config"
)

type DiskStats struct {
	UsageBytes uint64
	LimitBytes uint64
}

type FileDescriptorStats struct {
	Open  uint64
	Limit uint64
}

type GPUStats struct {
	Available          bool
	UtilizationRatio   float64
	MemoryUsedBytes    uint64
	MemoryTotalBytes   uint64
	TemperatureCelsius float64
	PowerWatts         float64
}

type NetworkStats struct {
	ReceiveBytesTotal    uint64
	TransmitBytesTotal   uint64
	ReceivePacketsTotal  uint64
	TransmitPacketsTotal uint64
	ReceiveErrorsTotal   uint64
	TransmitErrorsTotal  uint64
}

type SocketStats struct {
	TCPEstablished uint64
	TCPListen      uint64
	UDP            uint64
}

type WorkspaceSummary struct {
	CPUUsage    float64
	CPUSeconds  float64
	MemoryTotal uint64
	MemoryUsed  uint64
	MemoryRSS   uint64
	DiskTotal   uint64
	DiskUsed    uint64
	FDOpen      uint64
	FDLimit     uint64
	GPU         *GPUStats
}

var procRoot = "/proc"

func GetWorkspaceSummary(includeGPU bool) (*WorkspaceSummary, error) {
	cpuUsage, cpu, err := GetCPUUsagePercent()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU stats: %w", err)
	}

	mem, err := GetMemoryStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory stats: %w", err)
	}

	disk, err := GetDiskStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk stats: %w", err)
	}

	fd, err := GetFileDescriptorStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get file descriptor stats: %w", err)
	}

	summary := &WorkspaceSummary{
		CPUUsage:    cpuUsage,
		CPUSeconds:  cpu.UsageSeconds,
		MemoryTotal: mem.LimitBytes,
		MemoryUsed:  mem.UsageBytes,
		MemoryRSS:   mem.RSSBytes,
		DiskTotal:   disk.LimitBytes,
		DiskUsed:    disk.UsageBytes,
		FDOpen:      fd.Open,
		FDLimit:     fd.Limit,
	}

	if includeGPU {
		summary.GPU = GetGPUStats()
	}

	return summary, nil
}

func GetDiskStats() (*DiskStats, error) {
	path := config.MustResolve("server", "root")

	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, fmt.Errorf("failed to get disk stats for %s: %w", path, err)
	}

	total := stat.Blocks * uint64(stat.Bsize)

	return &DiskStats{UsageBytes: total - stat.Bavail*uint64(stat.Bsize), LimitBytes: total}, nil
}

func GetFileDescriptorStats() (*FileDescriptorStats, error) {
	dirs, err := filepath.Glob(filepath.Join(procRoot, "[0-9]*", "fd"))
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", procRoot, err)
	}

	stats := &FileDescriptorStats{Limit: readProcProperty(filepath.Join(procRoot, "self", "limits"), "Max open files", 3)}

	readable := 0
	for _, dir := range dirs {
		if entries, err := os.ReadDir(dir); err == nil {
			readable++
			stats.Open += uint64(len(entries))
		}
	}

	if readable == 0 {
		return nil, fmt.Errorf("no readable file descriptor tables under %s", procRoot)
	}

	return stats, nil
}

func IsGPUAvailable() bool {
	_, err := exec.LookPath("nvidia-smi")
	return err == nil
}

func GetGPUStats() *GPUStats {
	if !IsGPUAvailable() {
		return &GPUStats{}
	}

	out, err := exec.Command("nvidia-smi",
		"--query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu,power.draw",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return &GPUStats{}
	}

	return parseGPUStats(string(out))
}

func parseGPUStats(out string) *GPUStats {
	line, _, _ := strings.Cut(strings.TrimSpace(out), "\n")

	fields := strings.Split(line, ",")
	if len(fields) < 5 {
		return &GPUStats{}
	}

	return &GPUStats{
		Available:          true,
		UtilizationRatio:   atof(fields[0]) / 100,
		MemoryUsedBytes:    uint64(atof(fields[1]) * 1024 * 1024),
		MemoryTotalBytes:   uint64(atof(fields[2]) * 1024 * 1024),
		TemperatureCelsius: atof(fields[3]),
		PowerWatts:         atof(fields[4]),
	}
}

func GetNetworkStats() (*NetworkStats, error) {
	stats := &NetworkStats{}

	err := scanFields(filepath.Join(procRoot, "self", "net", "dev"), 2, func(fields []string) {
		_, counters, found := strings.Cut(strings.Join(fields, " "), ":")
		if fields = strings.Fields(counters); !found || len(fields) < 16 {
			return
		}

		stats.ReceiveBytesTotal += atoi(fields[0])
		stats.ReceivePacketsTotal += atoi(fields[1])
		stats.ReceiveErrorsTotal += atoi(fields[2])
		stats.TransmitBytesTotal += atoi(fields[8])
		stats.TransmitPacketsTotal += atoi(fields[9])
		stats.TransmitErrorsTotal += atoi(fields[10])
	})

	return stats, err
}

func GetSocketStats() (*SocketStats, error) {
	stats := &SocketStats{}

	for _, name := range []string{"tcp", "tcp6", "udp", "udp6"} {
		_ = scanFields(filepath.Join(procRoot, "self", "net", name), 1, func(fields []string) {
			if strings.HasPrefix(name, "udp") {
				stats.UDP++
				return
			}

			if len(fields) < 4 {
				return
			}

			switch state, _ := strconv.ParseUint(fields[3], 16, 8); state {
			case 1:
				stats.TCPEstablished++
			case 10:
				stats.TCPListen++
			}
		})
	}

	return stats, nil
}
