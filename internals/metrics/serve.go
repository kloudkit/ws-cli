package metrics

import (
	"errors"
	"slices"
	"strings"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/prometheus/client_golang/prometheus"
)

type RegistryResult struct {
	Registry *prometheus.Registry
	Expanded []string
	Invalid  []string
	Warnings []string
}

func BuildRegistry(collectors []string) (*RegistryResult, error) {
	result := &RegistryResult{}
	groups := collectorGroups()

	var leaves, validated []string
	for _, g := range groups {
		leaves = append(leaves, g.leaves()...)
	}

	for _, c := range collectors {
		if c = strings.TrimSpace(c); c == "" {
			continue
		}
		if c == "*" {
			validated = []string{"*"}
			break
		}
		if slices.ContainsFunc(leaves, func(leaf string) bool { return leaf == c || strings.HasPrefix(leaf, c+".") }) {
			validated = append(validated, c)
		} else {
			result.Invalid = append(result.Invalid, c)
		}
	}

	hasPressure, hasGPU := IsPressureAvailable(), IsGPUAvailable()
	isPressure := func(c string) bool { return c == "pressure" || strings.HasPrefix(c, "pressure.") }

	if !hasPressure && slices.ContainsFunc(validated, isPressure) {
		result.Warnings = append(result.Warnings, "PSI pressure metrics not available (cgroup v2 only), skipping pressure collector")
	}
	if !hasGPU && slices.Contains(validated, "gpu") {
		result.Warnings = append(result.Warnings, "GPU not available, skipping gpu collector")
	}

	enabled := func(leaf string) bool {
		return IsCollectorEnabled(leaf, validated) && (hasPressure || !isPressure(leaf)) && (hasGPU || leaf != "gpu")
	}

	result.Registry = prometheus.NewRegistry()
	for _, g := range groups {
		if c := g.only(enabled); c != nil {
			result.Registry.MustRegister(c)
		}
	}

	expanded := slices.DeleteFunc(leaves, func(leaf string) bool { return !enabled(leaf) })
	slices.Sort(expanded)
	result.Expanded = slices.Compact(expanded)
	if len(result.Expanded) == 0 {
		return nil, errors.New("no collectors enabled")
	}

	return result, nil
}

func DefaultPort() int {
	port, err := config.ResolveInt("metrics", "port")
	if err != nil {
		return 9100
	}
	return int(port)
}

func DefaultCollectors() []string {
	envCollectors, _ := config.Resolve("metrics", "collectors")
	if envCollectors == "" {
		return nil
	}

	var collectors []string
	for _, c := range strings.Split(envCollectors, ",") {
		if c = strings.TrimSpace(c); c != "" {
			collectors = append(collectors, c)
		}
	}
	return collectors
}
