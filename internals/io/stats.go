package io

import (
	"bufio"
	"fmt"
	"os"
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
	file, err := os.Open("/proc/self/limits")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Max open files") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				limit, _ := strconv.ParseUint(fields[4], 10, 64)
				return limit
			}
		}
	}

	return 0
}
