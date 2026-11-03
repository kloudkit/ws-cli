package metrics

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const cgroupRoot = "/sys/fs/cgroup"

type CPUStats struct {
	UsageSeconds     float64
	UserSeconds      float64
	SystemSeconds    float64
	ThrottledPeriods uint64
	ThrottledSeconds float64
	TotalPeriods     uint64
}

type MemoryStats struct {
	UsageBytes     uint64
	LimitBytes     uint64
	RSSBytes       uint64
	CacheBytes     uint64
	SwapBytes      uint64
	SwapLimitBytes uint64
	AnonBytes      uint64
	KernelBytes    uint64
	SlabBytes      uint64
	OOMEvents      uint64
	OOMKillEvents  uint64
	MaxEvents      uint64
}

type PIDStats struct {
	Current uint64
	Limit   uint64
}

type IOStats struct {
	ReadBytesTotal  uint64
	WriteBytesTotal uint64
	ReadOpsTotal    uint64
	WriteOpsTotal   uint64
}

type PressureStats struct {
	CPUWaitingSeconds    float64
	CPUStalledSeconds    float64
	MemoryWaitingSeconds float64
	MemoryStalledSeconds float64
	IOWaitingSeconds     float64
	IOStalledSeconds     float64
}

func GetCPUStats() (*CPUStats, error) {
	if cgroupsV2() {
		kv, err := parseKVStats(cgroupRoot + "/cpu.stat")
		if err != nil {
			return nil, fmt.Errorf("failed to read cpu.stat: %w", err)
		}

		return &CPUStats{
			UsageSeconds:     float64(kv["usage_usec"]) / 1e6,
			UserSeconds:      float64(kv["user_usec"]) / 1e6,
			SystemSeconds:    float64(kv["system_usec"]) / 1e6,
			TotalPeriods:     kv["nr_periods"],
			ThrottledPeriods: kv["nr_throttled"],
			ThrottledSeconds: float64(kv["throttled_usec"]) / 1e6,
		}, nil
	}

	var err error
	for _, dir := range []string{cgroupRoot + "/cpu", cgroupRoot + "/cpuacct"} {
		var usage uint64
		if usage, err = readUint64FromFile(dir + "/cpuacct.usage"); err == nil {
			kv, _ := parseKVStats(dir + "/cpuacct.stat")

			return &CPUStats{
				UsageSeconds:  float64(usage) / 1e9,
				UserSeconds:   float64(kv["user"]) / 100,
				SystemSeconds: float64(kv["system"]) / 100,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to read cpuacct.usage: %w", err)
}

func GetCPUUsagePercent() (float64, *CPUStats, error) {
	first, err := GetCPUStats()
	if err != nil {
		return 0, nil, err
	}

	start := time.Now()
	time.Sleep(100 * time.Millisecond)

	second, err := GetCPUStats()
	if err != nil {
		return 0, nil, err
	}

	usage := (second.UsageSeconds - first.UsageSeconds) / time.Since(start).Seconds() / float64(runtime.NumCPU()) * 100

	return min(max(usage, 0), 100), second, nil
}

func GetMemoryStats() (*MemoryStats, error) {
	if !cgroupsV2() {
		usage, err := readUint64FromFile(cgroupRoot + "/memory/memory.usage_in_bytes")
		if err != nil {
			return nil, fmt.Errorf("failed to read memory.usage_in_bytes: %w", err)
		}

		stats := &MemoryStats{UsageBytes: usage, RSSBytes: rssFromStatus()}
		if limit := readUint(cgroupRoot + "/memory/memory.limit_in_bytes"); limit < 1<<62 {
			stats.LimitBytes = limit
		}

		return stats, nil
	}

	usage, err := readUint64FromFile(cgroupRoot + "/memory.current")
	if err != nil {
		return nil, fmt.Errorf("failed to read memory.current: %w", err)
	}

	kv, _ := parseKVStats(cgroupRoot + "/memory.stat")
	events, _ := parseKVStats(cgroupRoot + "/memory.events")

	stats := &MemoryStats{
		UsageBytes:     usage,
		LimitBytes:     readUint(cgroupRoot + "/memory.max"),
		RSSBytes:       kv["anon"],
		AnonBytes:      kv["anon"],
		CacheBytes:     kv["file"],
		KernelBytes:    kv["kernel"],
		SlabBytes:      kv["slab"],
		SwapBytes:      readUint(cgroupRoot + "/memory.swap.current"),
		SwapLimitBytes: readUint(cgroupRoot + "/memory.swap.max"),
		OOMEvents:      events["oom"],
		OOMKillEvents:  events["oom_kill"],
		MaxEvents:      events["max"],
	}

	if stats.RSSBytes == 0 {
		stats.RSSBytes = rssFromStatus()
	}

	return stats, nil
}

func rssFromStatus() uint64 {
	return readProcProperty("/proc/self/status", "VmRSS:", 1) * 1024
}

func GetPIDStats() (*PIDStats, error) {
	dir := cgroupRoot + "/pids"
	if cgroupsV2() {
		dir = cgroupRoot
	}

	current, err := readUint64FromFile(dir + "/pids.current")
	if err != nil {
		return nil, fmt.Errorf("failed to read pids.current: %w", err)
	}

	return &PIDStats{Current: current, Limit: readUint(dir + "/pids.max")}, nil
}

func GetIOStats() (*IOStats, error) {
	stats := &IOStats{}

	if !cgroupsV2() {
		parseBlkIO(cgroupRoot+"/blkio/blkio.throttle.io_service_bytes", &stats.ReadBytesTotal, &stats.WriteBytesTotal)
		parseBlkIO(cgroupRoot+"/blkio/blkio.throttle.io_serviced", &stats.ReadOpsTotal, &stats.WriteOpsTotal)

		return stats, nil
	}

	counters := map[string]*uint64{
		"rbytes": &stats.ReadBytesTotal,
		"wbytes": &stats.WriteBytesTotal,
		"rios":   &stats.ReadOpsTotal,
		"wios":   &stats.WriteOpsTotal,
	}

	err := scanFields(cgroupRoot+"/io.stat", 0, func(fields []string) {
		for _, field := range fields[1:] {
			key, value, _ := strings.Cut(field, "=")
			if counter, ok := counters[key]; ok {
				*counter += atoi(value)
			}
		}
	})

	return stats, err
}

func parseBlkIO(path string, read, write *uint64) {
	_ = scanFields(path, 0, func(fields []string) {
		if len(fields) != 3 {
			return
		}

		switch fields[1] {
		case "Read":
			*read += atoi(fields[2])
		case "Write":
			*write += atoi(fields[2])
		}
	})
}

func GetPressureStats() (*PressureStats, error) {
	stats := &PressureStats{}

	stats.CPUWaitingSeconds, stats.CPUStalledSeconds = parsePressure(cgroupRoot + "/cpu.pressure")
	stats.MemoryWaitingSeconds, stats.MemoryStalledSeconds = parsePressure(cgroupRoot + "/memory.pressure")
	stats.IOWaitingSeconds, stats.IOStalledSeconds = parsePressure(cgroupRoot + "/io.pressure")

	return stats, nil
}

func IsPressureAvailable() bool {
	_, err := os.Stat(cgroupRoot + "/cpu.pressure")
	return err == nil
}

func parsePressure(path string) (some, full float64) {
	_ = scanFields(path, 0, func(fields []string) {
		var total float64
		for _, field := range fields[1:] {
			if value, found := strings.CutPrefix(field, "total="); found {
				total = float64(atoi(value)) / 1e6
			}
		}

		switch fields[0] {
		case "some":
			some = total
		case "full":
			full = total
		}
	})

	return some, full
}

var cgroupsV2 = sync.OnceValue(func() bool {
	_, err := os.Stat(cgroupRoot + "/cgroup.controllers")
	return err == nil
})

func readUint64FromFile(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}

func readUint(path string) uint64 {
	data, _ := os.ReadFile(path)
	return atoi(string(data))
}

func parseKVStats(path string) (map[string]uint64, error) {
	stats := make(map[string]uint64)
	err := scanFields(path, 0, func(fields []string) {
		if len(fields) >= 2 {
			if val, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
				stats[fields[0]] = val
			}
		}
	})
	return stats, err
}

func scanFields(path string, skip int, handler func(fields []string)) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for i := 0; scanner.Scan(); i++ {
		if fields := strings.Fields(scanner.Text()); i >= skip && len(fields) > 0 {
			handler(fields)
		}
	}
	return scanner.Err()
}

func atoi(s string) uint64 {
	v, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return v
}

func atof(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

func readProcProperty(path, prefix string, fieldIndex int) uint64 {
	var result uint64
	_ = scanFields(path, 0, func(fields []string) {
		if strings.HasPrefix(strings.Join(fields, " "), prefix) && len(fields) > fieldIndex {
			result = atoi(fields[fieldIndex])
		}
	})
	return result
}
