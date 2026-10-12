package metrics

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/kloudkit/ws-cli/internals/config"
)

func GetDiskStats() (*DiskStats, error) {
	path := config.MustResolve("server", "root")

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

var procRoot = "/proc"

func GetFileDescriptorStats() (*FileDescriptorStats, error) {
	dirs, err := filepath.Glob(filepath.Join(procRoot, "[0-9]*", "fd"))
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", procRoot, err)
	}

	stats := &FileDescriptorStats{
		Limit: readProcProperty(filepath.Join(procRoot, "self", "limits"), "Max open files", 4),
	}

	readable := 0
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		readable++
		stats.Open += uint64(len(entries))
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
		return &GPUStats{Available: false}
	}

	out, err := exec.Command("nvidia-smi",
		"--query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu,power.draw",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return &GPUStats{Available: false}
	}

	fields := strings.Split(strings.TrimSpace(string(out)), ", ")
	if len(fields) < 5 {
		return &GPUStats{Available: false}
	}

	stats := &GPUStats{Available: true}

	stats.UtilizationRatio = atof(fields[0]) / 100.0
	stats.MemoryUsedBytes = uint64(atof(fields[1]) * 1024 * 1024)
	stats.MemoryTotalBytes = uint64(atof(fields[2]) * 1024 * 1024)
	stats.TemperatureCelsius = atof(fields[3])
	stats.PowerWatts = atof(fields[4])

	return stats
}
