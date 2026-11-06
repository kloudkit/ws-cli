package features

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type StoreManifest struct {
	Built    string         `json:"built"`
	Packages []StorePackage `json:"packages"`
}

type StorePackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"`
}

func (m *StoreManifest) Artifacts() []StorePackage {
	var artifacts []StorePackage

	seen := map[string]bool{}
	for _, p := range m.Packages {
		if p.Type != "artifact" || seen[p.Name] {
			continue
		}

		seen[p.Name] = true
		artifacts = append(artifacts, p)
	}

	return artifacts
}

func FetchStoreManifest(baseURL string) (*StoreManifest, error) {
	manifestURL, err := url.JoinPath(baseURL, "manifest.json")
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
