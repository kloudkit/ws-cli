package metrics

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func GetCPUStats() (*CPUStats, error) {
	return isCgroupsV2(getCPUStatsV2, getCPUStatsV1)
}

func getCPUStatsV2() (*CPUStats, error) {
	kv, err := parseKVStats("/sys/fs/cgroup/cpu.stat")
	if err != nil {
		return nil, fmt.Errorf("failed to read cpu.stat: %w", err)
	}

	stats := &CPUStats{}
	stats.UsageSeconds = float64(kv["usage_usec"]) / 1e6
	stats.UserSeconds = float64(kv["user_usec"]) / 1e6
	stats.SystemSeconds = float64(kv["system_usec"]) / 1e6
	stats.TotalPeriods = kv["nr_periods"]
	stats.ThrottledPeriods = kv["nr_throttled"]
	stats.ThrottledSeconds = float64(kv["throttled_usec"]) / 1e6

	return stats, nil
}

func getCPUStatsV1() (*CPUStats, error) {
	stats := &CPUStats{}

	usageData, err := readUint64FromFile("/sys/fs/cgroup/cpu/cpuacct.usage")
	if err != nil {
		usageData, err = readUint64FromFile("/sys/fs/cgroup/cpuacct/cpuacct.usage")
		if err != nil {
			return nil, fmt.Errorf("failed to read cpuacct.usage: %w", err)
		}
	}
	stats.UsageSeconds = float64(usageData) / 1e9

	statPath := "/sys/fs/cgroup/cpu/cpuacct.stat"
	if _, err := os.Stat(statPath); err != nil {
		statPath = "/sys/fs/cgroup/cpuacct/cpuacct.stat"
	}

	kv, err := parseKVStats(statPath)
	if err == nil {
		userHz := 100.0
		stats.UserSeconds = float64(kv["user"]) / userHz
		stats.SystemSeconds = float64(kv["system"]) / userHz
	}

	return stats, nil
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

func GetMemoryStats() (*MemoryStats, error) {
	return isCgroupsV2(getMemoryStatsV2, getMemoryStatsV1)
}

func getMemoryStatsV2() (*MemoryStats, error) {
	stats := &MemoryStats{}
	var err error

	stats.UsageBytes, err = readUint64FromFile("/sys/fs/cgroup/memory.current")
	if err != nil {
		return nil, fmt.Errorf("failed to read memory.current: %w", err)
	}

	stats.LimitBytes = readCgroupLimit("/sys/fs/cgroup/memory.max")

	if kv, err := parseKVStats("/sys/fs/cgroup/memory.stat"); err == nil {
		stats.AnonBytes = kv["anon"]
		stats.CacheBytes = kv["file"]
		stats.KernelBytes = kv["kernel"]
		stats.SlabBytes = kv["slab"]
	}

	if stats.AnonBytes > 0 {
		stats.RSSBytes = stats.AnonBytes
	} else {
		stats.RSSBytes = getRSSFromStatus()
	}

	stats.SwapBytes, _ = readUint64FromFile("/sys/fs/cgroup/memory.swap.current")
	stats.SwapLimitBytes = readCgroupLimit("/sys/fs/cgroup/memory.swap.max")

	if kv, err := parseKVStats("/sys/fs/cgroup/memory.events"); err == nil {
		stats.OOMEvents = kv["oom"]
		stats.OOMKillEvents = kv["oom_kill"]
		stats.MaxEvents = kv["max"]
	}

	return stats, nil
}

func getMemoryStatsV1() (*MemoryStats, error) {
	stats := &MemoryStats{}
	var err error

	stats.UsageBytes, err = readUint64FromFile("/sys/fs/cgroup/memory/memory.usage_in_bytes")
	if err != nil {
		return nil, fmt.Errorf("failed to read memory.usage_in_bytes: %w", err)
	}

	if limit, err := readUint64FromFile("/sys/fs/cgroup/memory/memory.limit_in_bytes"); err == nil {
		if limit < 1<<62 {
			stats.LimitBytes = limit
		}
	}

	stats.RSSBytes = getRSSFromStatus()

	return stats, nil
}

func getRSSFromStatus() uint64 {
	return readProcProperty("/proc/self/status", "VmRSS:", 1) * 1024
}

func GetPIDStats() (*PIDStats, error) {
	return isCgroupsV2(getPIDStatsV2, getPIDStatsV1)
}

func getPIDStatsV2() (*PIDStats, error) {
	stats := &PIDStats{}
	var err error

	stats.Current, err = readUint64FromFile("/sys/fs/cgroup/pids.current")
	if err != nil {
		return nil, fmt.Errorf("failed to read pids.current: %w", err)
	}

	stats.Limit = readCgroupLimit("/sys/fs/cgroup/pids.max")

	return stats, nil
}

func getPIDStatsV1() (*PIDStats, error) {
	stats := &PIDStats{}
	var err error

	stats.Current, err = readUint64FromFile("/sys/fs/cgroup/pids/pids.current")
	if err != nil {
		return nil, fmt.Errorf("failed to read pids.current: %w", err)
	}

	stats.Limit = readCgroupLimit("/sys/fs/cgroup/pids/pids.max")

	return stats, nil
}

func GetIOStats() (*IOStats, error) {
	return isCgroupsV2(getIOStatsV2, getIOStatsV1)
}

func getIOStatsV2() (*IOStats, error) {
	stats := &IOStats{}
	err := processFileLines("/sys/fs/cgroup/io.stat", func(line string) {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return
		}

		for _, field := range fields[1:] {
			parts := strings.SplitN(field, "=", 2)
			if len(parts) != 2 {
				continue
			}

			value, err := strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				continue
			}

			switch parts[0] {
			case "rbytes":
				stats.ReadBytesTotal += value
			case "wbytes":
				stats.WriteBytesTotal += value
			case "rios":
				stats.ReadOpsTotal += value
			case "wios":
				stats.WriteOpsTotal += value
			}
		}
	})
	return stats, err
}

func getIOStatsV1() (*IOStats, error) {
	stats := &IOStats{}

	parseCgroupV1BlkIO("/sys/fs/cgroup/blkio/blkio.throttle.io_service_bytes", stats, true)
	parseCgroupV1BlkIO("/sys/fs/cgroup/blkio/blkio.throttle.io_serviced", stats, false)

	return stats, nil
}

func parseCgroupV1BlkIO(path string, stats *IOStats, isBytes bool) {
	_ = processFileLines(path, func(line string) {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return
		}

		value, err := strconv.ParseUint(fields[2], 10, 64)
		if err != nil {
			return
		}

		if isBytes {
			switch fields[1] {
			case "Read":
				stats.ReadBytesTotal += value
			case "Write":
				stats.WriteBytesTotal += value
			}
		} else {
			switch fields[1] {
			case "Read":
				stats.ReadOpsTotal += value
			case "Write":
				stats.WriteOpsTotal += value
			}
		}
	})
}

func GetPressureStats() (*PressureStats, error) {
	stats := &PressureStats{}

	cpuSome, cpuFull := parsePressureFile("/sys/fs/cgroup/cpu.pressure")
	stats.CPUWaitingSeconds = cpuSome
	stats.CPUStalledSeconds = cpuFull

	memSome, memFull := parsePressureFile("/sys/fs/cgroup/memory.pressure")
	stats.MemoryWaitingSeconds = memSome
	stats.MemoryStalledSeconds = memFull

	ioSome, ioFull := parsePressureFile("/sys/fs/cgroup/io.pressure")
	stats.IOWaitingSeconds = ioSome
	stats.IOStalledSeconds = ioFull

	return stats, nil
}

func IsPressureAvailable() bool {
	_, err := os.Stat("/sys/fs/cgroup/cpu.pressure")
	return err == nil
}

func parsePressureFile(path string) (some, full float64) {
	_ = processFileLines(path, func(line string) {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return
		}

		var totalUsec uint64
		for _, field := range fields[1:] {
			if value, found := strings.CutPrefix(field, "total="); found {
				totalUsec, _ = strconv.ParseUint(value, 10, 64)
				break
			}
		}

		switch fields[0] {
		case "some":
			some = float64(totalUsec) / 1e6
		case "full":
			full = float64(totalUsec) / 1e6
		}
	})

	return some, full
}
