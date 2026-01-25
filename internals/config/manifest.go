package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Manifest struct {
	Version string `json:"version"`
	VSCode  struct {
		Version string `json:"version"`
	} `json:"vscode"`
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
