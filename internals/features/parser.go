package features

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

type FeatureSource string

const (
	SourceSystem   FeatureSource = "system"
	SourceUser     FeatureSource = "user"
	SourceOverride FeatureSource = "override"
)

type Feature struct {
	Name        string
	Description string
	Vars        []string
	Source      FeatureSource
}

type ListResult struct {
	Features []*Feature
	Warnings []string
}

var jinjaExpr = regexp.MustCompile(`\s*\{\{.*?\}\}`)

func ParseFeatureFile(filePath string) (*Feature, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var plays []struct {
		Name string         `yaml:"name"`
		Vars map[string]any `yaml:"vars"`
	}
	if err := yaml.Unmarshal(data, &plays); err != nil {
		return nil, fmt.Errorf("failed to parse YAML from %s: %w", filePath, err)
	}

	if len(plays) == 0 {
		return nil, fmt.Errorf("no playbook tasks found in %s", filePath)
	}

	return &Feature{
		Name:        strings.TrimSuffix(filepath.Base(filePath), ".yaml"),
		Description: strings.TrimSpace(jinjaExpr.ReplaceAllString(plays[0].Name, "")),
		Vars:        slices.Sorted(maps.Keys(plays[0].Vars)),
	}, nil
}

func resolveOwner(dirs []string, name string) (string, FeatureSource, error) {
	owner, ownerIdx, present := "", -1, 0

	for idx, dir := range dirs {
		candidate := filepath.Join(dir, name+".yaml")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			owner, ownerIdx = candidate, idx
			present++
		}
	}

	switch {
	case owner == "":
		return "", "", fmt.Errorf("feature '%s' not found", name)
	case len(dirs) < 2 || ownerIdx < len(dirs)-1:
		return owner, SourceSystem, nil
	case present > 1:
		return owner, SourceOverride, nil
	default:
		return owner, SourceUser, nil
	}
}

func ResolveFeaturePath(dirs []string, name string) (string, error) {
	owner, _, err := resolveOwner(dirs, name)
	return owner, err
}

func InfoFeature(dirs []string, name string) (*Feature, error) {
	owner, source, err := resolveOwner(dirs, name)
	if err != nil {
		return nil, err
	}

	feature, err := ParseFeatureFile(owner)
	if err != nil {
		return nil, err
	}

	feature.Source = source

	return feature, nil
}

func ListFeatures(dirs []string) (*ListResult, error) {
	names := map[string]bool{}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read features directory %s: %w", dir, err)
		}

		for _, entry := range entries {
			if name, ok := strings.CutSuffix(entry.Name(), ".yaml"); ok && !entry.IsDir() {
				names[name] = true
			}
		}
	}

	result := &ListResult{}
	for _, name := range slices.Sorted(maps.Keys(names)) {
		feature, err := InfoFeature(dirs, name)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("skipped %s.yaml: %v", name, err))
			continue
		}

		result.Features = append(result.Features, feature)
	}

	return result, nil
}
