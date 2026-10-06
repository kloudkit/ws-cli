package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"

	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/kloudkit/ws-cli/internals/path"
)

type CheckState int

const (
	CheckPreferredSet CheckState = iota
	CheckDeprecatedOnly
	CheckBothSet
	CheckUnset
)

type ResolveSource int

const (
	SourceEnv ResolveSource = iota
	SourceDeprecatedAlias
	SourceDefault
	SourceEnvFile
	SourceSecretFileDefault
)

func (s ResolveSource) Label() string {
	switch s {
	case SourceEnv:
		return "process"
	case SourceDeprecatedAlias:
		return "alias"
	case SourceDefault:
		return "declared"
	case SourceEnvFile:
		return "file"
	case SourceSecretFileDefault:
		return "mount"
	}
	return ""
}

const defaultSecretConventionRoot = "/run/secrets/workspace"

func SecretConventionPath(group, prop string) string {
	root := env.String("WS__INTERNAL_SECRETS_ROOT", defaultSecretConventionRoot)
	return root + "/" + strings.ToLower(group) + "/" + strings.ToLower(prop)
}

func readSecretFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read secret file [%s]: %w", path, err)
	}
	return strings.TrimSuffix(string(data), "\n"), nil
}

func resolveSecretFromEnv(prop Property, value string) (string, ResolveSource, error) {
	file := strings.TrimPrefix(value, "file:")
	if file == "" {
		return "", SourceEnv, fmt.Errorf(
			"file: prefix requires a path [%s]", RuntimeKey(prop.Group, prop.Name),
		)
	}
	if prop.Type == "path" {
		file = path.ExpandHome(file)
	}
	contents, err := readSecretFile(file)
	if err != nil {
		return "", SourceEnv, err
	}
	return contents, SourceEnvFile, nil
}

var (
	deprecationWriter io.Writer = os.Stderr
	warnedAliases     sync.Map
)

func SetDeprecationWriter(w io.Writer) io.Writer {
	original := deprecationWriter
	deprecationWriter = w
	return original
}

func ResetWarnedAliases() {
	warnedAliases.Clear()
}

func Resolve(group, prop string) (string, error) {
	return ResolveKey(RuntimeKey(group, prop))
}

func ResolveInt(group, prop string) (int64, error) {
	v, err := Resolve(group, prop)
	if err != nil {
		return 0, err
	}
	return ParseInt(v)
}

func ResolveKey(runtimeKey string) (string, error) {
	value, _, err := ResolveKeyWithSource(runtimeKey)
	return value, err
}

func LookupProperty(runtimeKey string) (Property, bool, error) {
	ref, err := LoadEnvReference()
	if err != nil {
		return Property{}, false, err
	}
	prop, ok := ref.Properties[runtimeKey]
	return prop, ok, nil
}

func ResolveKeyWithSource(runtimeKey string) (string, ResolveSource, error) {
	ref, refErr := LoadEnvReference()
	value, source, err := resolveValueAndSource(ref, refErr, runtimeKey)
	if err != nil {
		return value, source, err
	}
	if ref != nil {
		if prop, ok := ref.Properties[runtimeKey]; ok {
			if prop.Type == "path" && source != SourceEnvFile && source != SourceSecretFileDefault {
				value = path.ExpandHome(value)
			}
			if err := prop.Validate(value); err != nil {
				return value, source, err
			}
		}
	}
	return value, source, nil
}

func resolveValueAndSource(ref *EnvReference, refErr error, runtimeKey string) (string, ResolveSource, error) {
	raw := env.String(runtimeKey)
	if refErr != nil {
		if raw != "" {
			return raw, SourceEnv, nil
		}
		return "", SourceDefault, refErr
	}
	prop, hasProp := ref.Properties[runtimeKey]

	if raw != "" {
		if strings.HasPrefix(raw, "file:") {
			if !hasProp || !prop.Secret {
				return "", SourceEnv, fmt.Errorf(
					"file: prefix is only valid on secret properties [%s]", runtimeKey,
				)
			}
			return resolveSecretFromEnv(prop, raw)
		}
		return raw, SourceEnv, nil
	}

	for _, alias := range ref.AliasesByPreferred[runtimeKey] {
		if v := env.String(alias); v != "" {
			emitDeprecationWarn(alias, runtimeKey)
			return v, SourceDeprecatedAlias, nil
		}
	}

	if hasProp && prop.Secret {
		contents, err := readSecretFile(SecretConventionPath(prop.Group, prop.Name))
		if err == nil {
			return contents, SourceSecretFileDefault, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", SourceSecretFileDefault, err
		}
	}

	if hasProp && prop.Default != nil {
		return *prop.Default, SourceDefault, nil
	}
	return "", SourceDefault, nil
}

func MustResolve(group, prop string) string {
	runtimeKey := RuntimeKey(group, prop)
	v, err := ResolveKey(runtimeKey)
	if err != nil {
		panic(fmt.Sprintf("config: %s: %v", runtimeKey, err))
	}
	return v
}

func ResolveListKey(runtimeKey, override string) ([]string, error) {
	ref, err := LoadEnvReference()
	if err != nil {
		return nil, err
	}
	prop, hasProp := ref.Properties[runtimeKey]
	delim := override
	if delim == "" {
		delim = prop.Delimiter
	}
	raw, _, err := resolveValueAndSource(ref, nil, runtimeKey)
	if err != nil {
		return nil, err
	}
	if hasProp {
		if err := prop.Validate(raw); err != nil {
			return nil, err
		}
	}
	items := ParseList(raw, delim)
	if hasProp && prop.Type == "path" {
		for i, item := range items {
			items[i] = path.ExpandHome(item)
		}
	}
	return items, nil
}

func Check(preferred, deprecated string) CheckState {
	preferredSet := env.String(preferred) != ""
	deprecatedSet := deprecated != "" && env.String(deprecated) != ""

	switch {
	case preferredSet && deprecatedSet:
		return CheckBothSet
	case preferredSet:
		return CheckPreferredSet
	case deprecatedSet:
		return CheckDeprecatedOnly
	}
	return CheckUnset
}

func emitDeprecationWarn(alias, preferred string) {
	if _, loaded := warnedAliases.LoadOrStore(alias, true); loaded {
		return
	}
	fmt.Fprintln(deprecationWriter, DeprecationLine(alias, preferred))
}

func DeprecationLine(alias, preferred string) string {
	return fmt.Sprintf("Deprecated: [%s] use [%s] instead", alias, preferred)
}

func BothSetLine(alias, preferred string) string {
	return fmt.Sprintf("Both [%s] (deprecated) and [%s] are set\n. Aborting", alias, preferred)
}
