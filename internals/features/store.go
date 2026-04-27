package features

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type StoreManifest struct {
	Built     string          `json:"built"`
	Artifacts []StoreArtifact `json:"artifacts"`
}

type StoreArtifact struct {
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Files   []string `json:"files"`
}

func FetchStoreManifest(baseURL string) (*StoreManifest, error) {
	manifestURL, err := url.JoinPath(baseURL, "artifacts", "manifest.json")
	if err != nil {
		return nil, fmt.Errorf("invalid store URL: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(manifestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to reach feature store: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feature store returned status %d", resp.StatusCode)
	}

	var manifest StoreManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}
