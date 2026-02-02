package metrics

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/kloudkit/ws-cli/internals/config"
)

func GetDiskStats() (*DiskStats, error) {
	return GetDiskStatsForPath(config.DefaultServerRoot)
}

func GetDiskStatsForPath(path string) (*DiskStats, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, fmt.Errorf("failed to get disk stats for %s: %w", path, err)
	}

	totalBytes := stat.Blocks * uint64(stat.Bsize)
	availBytes := stat.Bavail * uint64(stat.Bsize)

	return &DiskStats{
		UsageBytes: totalBytes - availBytes,
		LimitBytes: totalBytes,
	}, nil
}

func GetFileDescriptorStats() (*FileDescriptorStats, error) {
	stats := &FileDescriptorStats{}

	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/self/fd: %w", err)
	}
	stats.Open = uint64(len(entries))

	stats.Limit = getFDLimit()

	return stats, nil
}

func getFDLimit() uint64 {
	return readProcProperty("/proc/self/limits", "Max open files", 4)
}

func IsGPUAvailable() bool {
	_, err := exec.LookPath("nvidia-smi")
	return err == nil
}

func GetGPUStats() (*GPUStats, error) {
	if !IsGPUAvailable() {
		return &GPUStats{Available: false}, nil
	}

	out, err := exec.Command("nvidia-smi",
		"--query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu,power.draw",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return &GPUStats{Available: false}, nil
	}

	fields := strings.Split(strings.TrimSpace(string(out)), ", ")
	if len(fields) < 5 {
		return &GPUStats{Available: false}, nil
	}

	stats := &GPUStats{Available: true}

	stats.UtilizationRatio = atof(fields[0]) / 100.0
	stats.MemoryUsedBytes = uint64(atof(fields[1]) * 1024 * 1024)
	stats.MemoryTotalBytes = uint64(atof(fields[2]) * 1024 * 1024)
	stats.TemperatureCelsius = atof(fields[3])
	stats.PowerWatts = atof(fields[4])

	return stats, nil
}
