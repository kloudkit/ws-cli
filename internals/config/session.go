package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GetInitializedTime() (time.Time, error) {
	path := filepath.Join(DefaultStatePath, "initialized")
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read initialized file: %w", err)
	}

	parsedTime, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return parsedTime, nil
}

func GetUptime() (time.Duration, error) {
	initialized, err := GetInitializedTime()
	if err != nil {
		return 0, err
	}

	return time.Since(initialized), nil
}
