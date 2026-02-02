package metrics

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func isCgroupsV2[T any](v2, v1 func() (T, error)) (T, error) {
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
		return v2()
	}
	return v1()
}

func readUint64FromFile(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}

func readCgroupLimit(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(data))
	if s == "max" {
		return 0
	}
	return atoi(s)
}

func parseKVStats(path string) (map[string]uint64, error) {
	stats := make(map[string]uint64)
	err := processFileLines(path, func(line string) {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			val, err := strconv.ParseUint(fields[1], 10, 64)
			if err == nil {
				stats[fields[0]] = val
			}
		}
	})
	return stats, err
}

func processFileLines(path string, handler func(line string)) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		handler(scanner.Text())
	}
	return scanner.Err()
}

func atoi(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

func atof(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func readProcProperty(path, prefix string, fieldIndex int) uint64 {
	var result uint64

	_ = processFileLines(path, func(line string) {
		if strings.HasPrefix(line, prefix) {
			fields := strings.Fields(line)
			if len(fields) > fieldIndex {
				result = atoi(fields[fieldIndex])
			}
		}
	})

	return result
}
