package config

import (
	"encoding/json"
	"fmt"
	"os"
)

func WorkspaceVersion() string {
	data, err := os.ReadFile(DefaultManifestPath)
	if err != nil {
		return ""
	}

	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ""
	}

	return manifest.Version
}

func VSCodeVersion() (string, error) {
	data, err := os.ReadFile(DefaultProductJSONPath)
	if err != nil {
		return "", fmt.Errorf("failed to read product.json: %w", err)
	}

	var product struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &product); err != nil {
		return "", fmt.Errorf("failed to parse product.json: %w", err)
	}

	return product.Version, nil
}
