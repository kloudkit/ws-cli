package io

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type CPUStats struct {
	UsageSeconds  float64
	UserSeconds   float64
	SystemSeconds float64
}

type MemoryStats struct {
	UsageBytes uint64
	LimitBytes uint64
	RSSBytes   uint64
}

func isCgroupsV2() bool {
	_, err := os.Stat("/sys/fs/cgroup/cgroup.controllers")

	return err == nil
}

func GetCPUStats() (*CPUStats, error) {
	if isCgroupsV2() {
		return getCPUStatsV2()
	}
	return getCPUStatsV1()
}

func getCPUStatsV2() (*CPUStats, error) {
	data, err := os.ReadFile("/sys/fs/cgroup/cpu.stat")
	if err != nil {
		return nil, fmt.Errorf("failed to read cpu.stat: %w", err)
	}

	stats := &CPUStats{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			continue
		}

		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		switch fields[0] {
		case "usage_usec":
			stats.UsageSeconds = float64(value) / 1e6
		case "user_usec":
			stats.UserSeconds = float64(value) / 1e6
		case "system_usec":
			stats.SystemSeconds = float64(value) / 1e6
		}
	}

	return stats, nil
}

func getCPUStatsV1() (*CPUStats, error) {
	stats := &CPUStats{}

	usageData, err := os.ReadFile("/sys/fs/cgroup/cpu/cpuacct.usage")
	if err != nil {
		usageData, err = os.ReadFile("/sys/fs/cgroup/cpuacct/cpuacct.usage")
		if err != nil {
			return nil, fmt.Errorf("failed to read cpuacct.usage: %w", err)
		}
	}

	usageNs, err := strconv.ParseUint(strings.TrimSpace(string(usageData)), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cpu usage: %w", err)
	}
	stats.UsageSeconds = float64(usageNs) / 1e9

	statData, err := os.ReadFile("/sys/fs/cgroup/cpu/cpuacct.stat")
	if err != nil {
		statData, err = os.ReadFile("/sys/fs/cgroup/cpuacct/cpuacct.stat")
		if err != nil {
			return stats, nil
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(string(statData)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			continue
		}

		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		userHz := uint64(100)
		switch fields[0] {
		case "user":
			stats.UserSeconds = float64(value) / float64(userHz)
		case "system":
			stats.SystemSeconds = float64(value) / float64(userHz)
		}
	}

	return stats, nil
}

func GetMemoryStats() (*MemoryStats, error) {
	if isCgroupsV2() {
		return getMemoryStatsV2()
	}

	return getMemoryStatsV1()
}

func getMemoryStatsV2() (*MemoryStats, error) {
	stats := &MemoryStats{}

	currentData, err := os.ReadFile("/sys/fs/cgroup/memory.current")
	if err != nil {
		return nil, fmt.Errorf("failed to read memory.current: %w", err)
	}
	stats.UsageBytes, _ = strconv.ParseUint(strings.TrimSpace(string(currentData)), 10, 64)

	maxData, err := os.ReadFile("/sys/fs/cgroup/memory.max")
	if err == nil {
		maxStr := strings.TrimSpace(string(maxData))
		if maxStr != "max" {
			stats.LimitBytes, _ = strconv.ParseUint(maxStr, 10, 64)
		}
	}

	stats.RSSBytes = getRSSFromStatus()

	return stats, nil
}

func getMemoryStatsV1() (*MemoryStats, error) {
	stats := &MemoryStats{}

	usageData, err := os.ReadFile("/sys/fs/cgroup/memory/memory.usage_in_bytes")
	if err != nil {
		return nil, fmt.Errorf("failed to read memory.usage_in_bytes: %w", err)
	}
	stats.UsageBytes, _ = strconv.ParseUint(strings.TrimSpace(string(usageData)), 10, 64)

	limitData, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes")
	if err == nil {
		limit, _ := strconv.ParseUint(strings.TrimSpace(string(limitData)), 10, 64)
		if limit < 1<<62 {
			stats.LimitBytes = limit
		}
	}

	stats.RSSBytes = getRSSFromStatus()

	return stats, nil
}

func getRSSFromStatus() uint64 {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				value, _ := strconv.ParseUint(fields[1], 10, 64)
				return value * 1024
			}
		}
	}

	return 0
}

func GetCPUUsagePercent() (float64, error) {
	stats1, err := GetCPUStats()
	if err != nil {
		return 0, err
	}

	time.Sleep(100 * time.Millisecond)

	stats2, err := GetCPUStats()
	if err != nil {
		return 0, err
	}

	cpuDelta := stats2.UsageSeconds - stats1.UsageSeconds
	timeDelta := 0.1

	numCPU := float64(runtime.NumCPU())
	usage := (cpuDelta / timeDelta / numCPU) * 100

	if usage > 100 {
		usage = 100
	}
	if usage < 0 {
		usage = 0
	}

	return usage, nil
}
