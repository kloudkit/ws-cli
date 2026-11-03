package config

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/kloudkit/ws-cli/internals/env"
	"go.yaml.in/yaml/v3"
)

type Property struct {
	Type            string  `yaml:"type"`
	Default         *string `yaml:"-"`
	Delimiter       string  `yaml:"delimiter"`
	Pattern         string  `yaml:"pattern"`
	Description     string  `yaml:"description"`
	LongDescription string  `yaml:"longDescription"`
	Secret          bool    `yaml:"secret"`
	Group           string  `yaml:"-"`
	Name            string  `yaml:"-"`
}

type Deprecation struct {
	Use     string `yaml:"use"`
	Message string `yaml:"message"`
	Removed string `yaml:"removed"`
}

type EnvReference struct {
	Properties         map[string]Property
	Deprecations       map[string]Deprecation
	AliasesByPreferred map[string][]string
}

var propertyTypes = []string{"string", "boolean", "integer", "path"}

func RuntimeKey(group, prop string) string {
	return "WS_" + strings.ToUpper(group) + "_" + strings.ToUpper(prop)
}

func LoadEnvReference() (*EnvReference, error) {
	path := env.String("WS__INTERNAL_ENV_REFERENCE", DefaultEnvReferencePath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read [%s]: %w", path, err)
	}
	return parseEnvReference(data)
}

func parseEnvReference(data []byte) (*EnvReference, error) {
	var raw struct {
		Envs map[string]struct {
			Properties map[string]struct {
				Property `yaml:",inline"`
				Default  any `yaml:"default"`
			} `yaml:"properties"`
		} `yaml:"envs"`
		Deprecated map[string]Deprecation `yaml:"deprecated"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("cannot parse env reference: %w", err)
	}

	ref := &EnvReference{
		Properties:         map[string]Property{},
		Deprecations:       raw.Deprecated,
		AliasesByPreferred: map[string][]string{},
	}

	for groupKey, group := range raw.Envs {
		for propKey, prop := range group.Properties {
			fail := func(format string, args ...any) error {
				return fmt.Errorf("env reference [%s.%s]: "+format, append([]any{groupKey, propKey}, args...)...)
			}
			if !slices.Contains(propertyTypes, prop.Type) {
				return nil, fail("unknown type [%s]", prop.Type)
			}

			var def *string
			if prop.Default != nil {
				s, isString := prop.Default.(string)
				if !isString && (prop.Type == "path" || prop.Secret) {
					constraint := "type [path]"
					if prop.Secret {
						constraint = "secret"
					}
					return nil, fail("%s requires default to be null or string, got %T", constraint, prop.Default)
				}
				if prop.Secret && strings.HasPrefix(s, "file:") {
					return nil, fail("secret properties cannot declare a [file:] default literal")
				}
				s = fmt.Sprint(prop.Default)
				def = &s
			}

			property := prop.Property
			property.Default, property.Group, property.Name = def, groupKey, propKey
			ref.Properties[RuntimeKey(groupKey, propKey)] = property
		}
	}

	for alias, dep := range ref.Deprecations {
		if dep.Removed != "" {
			continue
		}
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
	for current := target; current != ""; current = deprecations[current].Use {
		if seen[current] {
			return "", fmt.Errorf("deprecation cycle through [%s]", current)
		}
		seen[current] = true
		if _, isAlias := deprecations[current]; !isAlias {
			return current, nil
		}
	}
	return "", nil
}

func (p Property) Validate(value string) error {
	if p.Secret || value == "" {
		return nil
	}

	key := RuntimeKey(p.Group, p.Name)

	var err error
	switch p.Type {
	case "integer":
		_, err = ParseInt(value)
	case "boolean":
		_, err = ParseBool(value)
	case "path":
		if strings.IndexFunc(value, unicode.IsControl) >= 0 {
			err = fmt.Errorf("path contains a control character")
		}
	}
	if err != nil {
		return fmt.Errorf("env [%s]: %w", key, err)
	}

	if p.Pattern == "" {
		return nil
	}

	re, err := regexp.Compile("^(?:" + p.Pattern + ")$")
	if err != nil {
		return fmt.Errorf("env [%s]: invalid declared pattern %q: %w", key, p.Pattern, err)
	}

	for _, token := range ParseList(value, p.Delimiter) {
		if !re.MatchString(token) {
			return fmt.Errorf("env [%s]: value [%s] rejected by pattern", key, token)
		}
	}

	return nil
}

func ParseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	}
	return false, fmt.Errorf("not a boolean: %q (accepted: 1/true/yes/on or 0/false/no/off)", s)
}

func ParseInt(s string) (int64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("not an integer: %q", s)
	}
	return n, nil
}

func ParseList(s, delim string) []string {
	if s == "" {
		return nil
	}
	if delim == "" {
		delim = " "
	}
	out := []string{}
	for p := range strings.SplitSeq(s, delim) {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
