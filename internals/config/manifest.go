package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Manifest struct {
	Version string `json:"version"`
}

func ReadManifest() (*Manifest, error) {
	data, err := os.ReadFile(DefaultManifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("manifest not found at %s", DefaultManifestPath)
		}
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &m, nil
}

func WorkspaceVersion() string {
	m, err := ReadManifest()
	if err != nil {
		return ""
	}

	return m.Version
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
