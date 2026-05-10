package config

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/kloudkit/ws-cli/internals/env"
)

type CheckState int

const (
	CheckPreferredSet CheckState = iota
	CheckDeprecatedOnly
	CheckBothSet
	CheckUnset
)

var (
	deprecationWriter io.Writer = os.Stderr
	warnedAliases     sync.Map
)

func Resolve(group, prop string) (string, error) {
	return ResolveKey(RuntimeKey(group, prop))
}

func ResolveBool(group, prop string) (bool, error) {
	v, err := Resolve(group, prop)
	if err != nil {
		return false, err
	}
	return ParseBool(v)
}

func ResolveInt(group, prop string) (int64, error) {
	v, err := Resolve(group, prop)
	if err != nil {
		return 0, err
	}
	return ParseInt(v)
}

func ResolveList(group, prop, override string) ([]string, error) {
	return ResolveListKey(RuntimeKey(group, prop), override)
}

func ResolveKey(runtimeKey string) (string, error) {
	if v := env.String(runtimeKey); v != "" {
		return v, nil
	}
	ref, err := LoadEnvReference()
	if err != nil {
		return "", err
	}
	return ref.Resolve(runtimeKey), nil
}

func MustResolve(group, prop string) string {
	return MustResolveKey(RuntimeKey(group, prop))
}

func MustResolveKey(runtimeKey string) string {
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
	delim := override
	if delim == "" {
		delim = ref.Properties[runtimeKey].Delimiter
	}
	return ParseList(ref.Resolve(runtimeKey), delim), nil
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

func (r *EnvReference) Resolve(key string) string {
	if v := env.String(key); v != "" {
		return v
	}
	for _, alias := range r.AliasesByPreferred[key] {
		if v := env.String(alias); v != "" {
			emitDeprecationWarn(alias, key)
			return v
		}
	}
	if prop, ok := r.Properties[key]; ok && prop.Default != nil {
		return *prop.Default
	}
	return ""
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
