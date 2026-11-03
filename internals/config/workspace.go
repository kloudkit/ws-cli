package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultEnvReferencePath = "/etc/workspace/env.reference.yaml"
	DefaultStatePath        = "/var/lib/workspace/state"
)

var (
	DefaultManifestPath    = "/var/lib/workspace/manifest.json"
	DefaultProductJSONPath = "/usr/lib/workspace/vscode/product.json"
)

func Bootstrap() error {
	if info, err := os.Stat(DefaultManifestPath); err != nil || info.IsDir() {
		return fmt.Errorf("this command requires a running Kloud Workspace")
	}

	_, err := LoadEnvReference()

	return err
}

func readVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var doc struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", err
	}

	return doc.Version, nil
}

func WorkspaceVersion() string {
	version, _ := readVersion(DefaultManifestPath)

	return version
}

func VSCodeVersion() (string, error) {
	version, err := readVersion(DefaultProductJSONPath)
	if err != nil {
		return "", fmt.Errorf("failed to read product.json: %w", err)
	}

	return version, nil
}

func GetInitializedTime() (time.Time, error) {
	data, err := os.ReadFile(filepath.Join(DefaultStatePath, "initialized"))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read initialized file: %w", err)
	}

	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return parsed, nil
}

func GetUptime() (time.Duration, error) {
	initialized, err := GetInitializedTime()
	if err != nil {
		return 0, err
	}

	return time.Since(initialized), nil
}
