package features

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
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

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no playbook tasks found in %s", filePath)
	}

	baseName := filepath.Base(filePath)

	var vars []string
	if tasks[0].Vars != nil {
		for key := range tasks[0].Vars {
			vars = append(vars, key)
		}
	}
	slices.Sort(vars)

	return &Feature{
		Name:        strings.TrimSuffix(baseName, ".yaml"),
		Description: tasks[0].Name,
		Vars:        vars,
	}, nil
}

func resolveOwner(dirs []string, name string) (ownerPath string, ownerIdx, presentCount int) {
	ownerIdx = -1

	for idx, dir := range dirs {
		candidate := filepath.Join(dir, name+".yaml")

		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}

		ownerPath = candidate
		ownerIdx = idx
		presentCount++
	}

	return ownerPath, ownerIdx, presentCount
}

func sourceFor(ownerIdx, presentCount, numDirs int) FeatureSource {
	if numDirs < 2 || ownerIdx < numDirs-1 {
		return SourceSystem
	}

	if presentCount > 1 {
		return SourceOverride
	}

	return SourceUser
}

func InfoFeature(dirs []string, name string) (*Feature, error) {
	ownerPath, ownerIdx, presentCount := resolveOwner(dirs, name)
	if ownerPath == "" {
		return nil, fmt.Errorf("feature '%s' not found", name)
	}

	feature, err := ParseFeatureFile(ownerPath)
	if err != nil {
		return nil, err
	}

	feature.Source = sourceFor(ownerIdx, presentCount, len(dirs))

	return feature, nil
}

func ResolveFeaturePath(dirs []string, name string) (string, error) {
	ownerPath, _, _ := resolveOwner(dirs, name)
	if ownerPath == "" {
		return "", fmt.Errorf("feature '%s' not found", name)
	}

	return ownerPath, nil
}

type ListResult struct {
	Features []*Feature
	Warnings []string
}

func ListFeatures(dirs []string) (*ListResult, error) {
	owners := map[string]string{}
	ownerIdx := map[string]int{}
	presentCount := map[string]int{}
	names := []string{}

	for idx, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read features directory %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}

			name := strings.TrimSuffix(entry.Name(), ".yaml")
			if _, seen := owners[name]; !seen {
				names = append(names, name)
			}

			owners[name] = filepath.Join(dir, entry.Name())
			ownerIdx[name] = idx
			presentCount[name]++
		}
	}

	slices.Sort(names)

	result := &ListResult{}
	for _, name := range names {
		feature, err := ParseFeatureFile(owners[name])
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("skipped %s: %v", filepath.Base(owners[name]), err))
			continue
		}

		feature.Source = sourceFor(ownerIdx[name], presentCount[name], len(dirs))
		result.Features = append(result.Features, feature)
	}

	return result, nil
}
