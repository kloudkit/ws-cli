package config

import (
	"fmt"
	"io"
	"os"
	"sync"
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

func ResolveListKey(runtimeKey, override string) ([]string, error) {
	ref, err := LoadEnvReference()
	if err != nil {
		return nil, err
	}
	v, err := ref.Resolve(runtimeKey)
	if err != nil {
		return nil, err
	}
	delim := override
	if delim == "" {
		delim = ref.Properties[runtimeKey].Delimiter
	}
	if delim == "" {
		delim = " "
	}
	return ParseList(v, delim), nil
}

func ResolveKey(runtimeKey string) (string, error) {
	ref, err := LoadEnvReference()
	if err != nil {
		return "", err
	}
	return ref.Resolve(runtimeKey)
}

func Check(preferred, deprecated string) CheckState {
	if v, ok := os.LookupEnv(preferred); ok && v != "" {
		if deprecated != "" {
			if dv, dok := os.LookupEnv(deprecated); dok && dv != "" {
				return CheckBothSet
			}
		}
		return CheckPreferredSet
	}
	if deprecated != "" {
		if dv, dok := os.LookupEnv(deprecated); dok && dv != "" {
			return CheckDeprecatedOnly
		}
	}
	return CheckUnset
}

func (r *EnvReference) Resolve(key string) (string, error) {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v, nil
	}

	for _, alias := range r.AliasesByPreferred[key] {
		if v, ok := os.LookupEnv(alias); ok && v != "" {
			emitDeprecationWarn(alias, key)
			return v, nil
		}
	}

	if prop, ok := r.Properties[key]; ok && prop.Default != nil {
		return *prop.Default, nil
	}

	return "", nil
}

func emitDeprecationWarn(alias, preferred string) {
	if _, loaded := warnedAliases.LoadOrStore(alias, true); loaded {
		return
	}
	fmt.Fprintf(deprecationWriter, "Deprecated: [%s] use [%s] instead\n", alias, preferred)
}

func DeprecationLine(alias, preferred string) string {
	return fmt.Sprintf("Deprecated: [%s] use [%s] instead", alias, preferred)
}

func BothSetLine(alias, preferred string) string {
	return fmt.Sprintf("Both [%s] (deprecated) and [%s] are set\n. Aborting", alias, preferred)
}
