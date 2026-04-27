package metrics

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/env"
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

	var validated []string
	hasExplicit := len(collectors) > 0

	for _, c := range collectors {
		if c = strings.TrimSpace(c); c == "" {
			continue
		}
		if c == "*" {
			validated = []string{"*"}
			break
		}
		if _, ok := ValidCollectors[c]; ok {
			validated = append(validated, c)
		} else {
			result.Invalid = append(result.Invalid, c)
		}
	}

	hasWorkspace := IsCollectorEnabled("workspace", validated)
	hasContainer := IsCollectorEnabled("container", validated)
	hasPressure := IsCollectorEnabled("pressure", validated) && IsPressureAvailable()
	hasNetwork := IsCollectorEnabled("network", validated)
	hasIO := IsCollectorEnabled("io", validated)
	hasSockets := IsCollectorEnabled("sockets", validated)
	gpuRequested := slices.Contains(validated, "gpu")
	hasGPU := IsCollectorEnabled("gpu", validated) && IsGPUAvailable()

	pressureRequested := slices.Contains(validated, "pressure") || slices.Contains(validated, "pressure.cpu") || slices.Contains(validated, "pressure.memory") || slices.Contains(validated, "pressure.io")
	if pressureRequested && !IsPressureAvailable() {
		result.Warnings = append(result.Warnings, "PSI pressure metrics not available (cgroup v2 only), skipping pressure collector")
		validated = slices.DeleteFunc(validated, func(c string) bool {
			return c == "pressure" || strings.HasPrefix(c, "pressure.")
		})
		hasPressure = false
	}

	if gpuRequested && !hasGPU {
		result.Warnings = append(result.Warnings, "GPU not available, skipping gpu collector")
		validated = slices.DeleteFunc(validated, func(c string) bool { return c == "gpu" })
	}

	if hasExplicit && !hasWorkspace && !hasContainer && !hasPressure && !hasNetwork && !hasIO && !hasSockets && !hasGPU {
		return nil, errors.New("no collectors enabled")
	}

	registry := prometheus.NewRegistry()
	if hasWorkspace {
		registry.MustRegister(NewWorkspaceCollector(validated))
	}
	if hasContainer {
		registry.MustRegister(NewContainerCollector(validated))
	}
	if hasPressure {
		registry.MustRegister(NewPressureCollector(validated))
	}
	if hasNetwork {
		registry.MustRegister(NewNetworkCollector())
	}
	if hasIO {
		registry.MustRegister(NewIOCollector())
	}
	if hasSockets {
		registry.MustRegister(NewSocketsCollector())
	}
	if hasGPU {
		registry.MustRegister(NewGPUCollector())
	}

	result.Registry = registry
	result.Expanded = ExpandCollectors(validated)
	return result, nil
}

func DefaultPort() int {
	if portStr := env.String(config.EnvMetricsPort); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			return port
		}
	}
	return config.DefaultMetricsPort
}

func DefaultCollectors() []string {
	envCollectors := env.String(config.EnvMetricsCollectors)
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
