package features

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Feature struct {
	Name        string
	Description string
	Vars        []string
}

type PlaybookTask struct {
	Name string         `yaml:"name"`
	Vars map[string]any `yaml:"vars"`
}

func ParseFeatureFile(filePath string) (*Feature, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var tasks []PlaybookTask
	if err := yaml.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse YAML from %s: %w", filePath, err)
	}

	baseName := filepath.Base(filePath)

	var vars []string
	if tasks[0].Vars != nil {
		for key := range tasks[0].Vars {
			vars = append(vars, key)
		}
	}

	return &Feature{
		Name:        strings.TrimSuffix(baseName, ".yaml"),
		Description: tasks[0].Name,
		Vars:        vars,
	}, nil
}

func InfoFeature(featuresDir, name string) (*Feature, error) {
	featurePath := filepath.Join(featuresDir, name+".yaml")

	if _, err := os.Stat(featurePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("feature '%s' not found at %s", name, featurePath)
	}

	return ParseFeatureFile(featurePath)
}

func ListFeatures(featuresDir string) ([]*Feature, error) {
	entries, err := os.ReadDir(featuresDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read features directory %s: %w", featuresDir, err)
	}

	var features []*Feature
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		feature, err := ParseFeatureFile(filepath.Join(featuresDir, entry.Name()))
		if err != nil {
			continue
		}

		features = append(features, feature)
	}

	return features, nil
}
