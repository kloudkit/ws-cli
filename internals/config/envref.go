package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const defaultEnvReferencePath = "/etc/workspace/env.reference.yaml"

type Property struct {
	Type      string
	Default   *string
	Delimiter string
}

type Deprecation struct {
	Use     string
	Message string
}

type EnvReference struct {
	Properties         map[string]Property
	Deprecations       map[string]Deprecation
	AliasesByPreferred map[string][]string
}

func RuntimeKey(group, prop string) string {
	return "WS_" + strings.ToUpper(group) + "_" + strings.ToUpper(prop)
}

type yamlSchema struct {
	Envs       map[string]yamlGroup       `yaml:"envs"`
	Deprecated map[string]yamlDeprecation `yaml:"deprecated"`
}

type yamlGroup struct {
	Properties map[string]yamlProperty `yaml:"properties"`
}

type yamlProperty struct {
	Type      string `yaml:"type"`
	Default   any    `yaml:"default"`
	Delimiter string `yaml:"delimiter"`
}

type yamlDeprecation struct {
	Use     string `yaml:"use"`
	Message string `yaml:"message"`
}

var (
	cacheMu    sync.Mutex
	cachedPath string
	cachedVal  *EnvReference
	cachedErr  error
)

func envReferencePath() string {
	if v := os.Getenv("WS__INTERNAL_ENV_REFERENCE"); v != "" {
		return v
	}
	return defaultEnvReferencePath
}

func LoadEnvReference() (*EnvReference, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	path := envReferencePath()
	if cachedVal != nil && cachedPath == path {
		return cachedVal, cachedErr
	}
	cachedPath = path
	cachedVal, cachedErr = readEnvReference(path)
	return cachedVal, cachedErr
}

func readEnvReference(path string) (*EnvReference, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read [%s]: %w", path, err)
	}
	return parseEnvReference(data)
}

func parseEnvReference(data []byte) (*EnvReference, error) {
	var raw yamlSchema
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("cannot parse env reference: %w", err)
	}

	ref := &EnvReference{
		Properties:         map[string]Property{},
		Deprecations:       map[string]Deprecation{},
		AliasesByPreferred: map[string][]string{},
	}

	for groupKey, group := range raw.Envs {
		groupUpper := strings.ToUpper(groupKey)
		for propKey, prop := range group.Properties {
			runtimeKey := "WS_" + groupUpper + "_" + strings.ToUpper(propKey)
			ref.Properties[runtimeKey] = Property{
				Type:      prop.Type,
				Default:   defaultFromAny(prop.Default),
				Delimiter: prop.Delimiter,
			}
		}
	}

	for alias, dep := range raw.Deprecated {
		ref.Deprecations[alias] = Deprecation{
			Use:     dep.Use,
			Message: dep.Message,
		}
	}

	for alias, dep := range ref.Deprecations {
		canonical, err := resolveCanonical(alias, dep.Use, ref.Deprecations)
		if err != nil {
			return nil, err
		}
		if canonical != "" {
			ref.AliasesByPreferred[canonical] = append(ref.AliasesByPreferred[canonical], alias)
		}
	}

	return ref, nil
}

func resolveCanonical(start, target string, deprecations map[string]Deprecation) (string, error) {
	seen := map[string]bool{start: true}
	current := target
	for current != "" {
		if seen[current] {
			return "", fmt.Errorf("deprecation cycle through [%s]", current)
		}
		seen[current] = true
		next, isAlias := deprecations[current]
		if !isAlias {
			return current, nil
		}
		current = next.Use
	}
	return "", nil
}

func defaultFromAny(v any) *string {
	if v == nil {
		return nil
	}
	s := fmt.Sprint(v)
	return &s
}
