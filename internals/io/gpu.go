package io

import (
	"os/exec"
	"strconv"
	"strings"
)

type GPUStats struct {
	Available          bool
	UtilizationRatio   float64
	MemoryUsedBytes    uint64
	MemoryTotalBytes   uint64
	TemperatureCelsius float64
	PowerWatts         float64
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

	if util, err := strconv.ParseFloat(strings.TrimSpace(fields[0]), 64); err == nil {
		stats.UtilizationRatio = util / 100.0
	}

	if memUsed, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64); err == nil {
		stats.MemoryUsedBytes = uint64(memUsed * 1024 * 1024)
	}

	if memTotal, err := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64); err == nil {
		stats.MemoryTotalBytes = uint64(memTotal * 1024 * 1024)
	}

	if temp, err := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64); err == nil {
		stats.TemperatureCelsius = temp
	}

	if power, err := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64); err == nil {
		stats.PowerWatts = power
	}

	return stats, nil
}
