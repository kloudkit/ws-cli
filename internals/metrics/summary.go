package metrics

import "fmt"

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

func GetWorkspaceSummary(includeGPU bool) (*WorkspaceSummary, error) {
	cpuUsage, _ := GetCPUUsagePercent()

	cpuStats, err := GetCPUStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU stats: %w", err)
	}

	memStats, err := GetMemoryStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory stats: %w", err)
	}

	diskStats, err := GetDiskStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk stats: %w", err)
	}

	fdStats, err := GetFileDescriptorStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get file descriptor stats: %w", err)
	}

	m := &WorkspaceSummary{
		CPUUsage:    cpuUsage,
		CPUSeconds:  cpuStats.UsageSeconds,
		MemoryTotal: memStats.LimitBytes,
		MemoryUsed:  memStats.UsageBytes,
		MemoryRSS:   memStats.RSSBytes,
		DiskTotal:   diskStats.LimitBytes,
		DiskUsed:    diskStats.UsageBytes,
		FDOpen:      fdStats.Open,
		FDLimit:     fdStats.Limit,
	}

	if includeGPU {
		if gpuStats, err := GetGPUStats(); err == nil {
			m.GPU = gpuStats
		}
	}

	return m, nil
}
