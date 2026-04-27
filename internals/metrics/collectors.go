package metrics

import (
	"slices"
	"strings"
)

const Namespace = "workspace"

var ValidCollectors = map[string][]string{
	"workspace":            {"workspace.info", "workspace.session", "workspace.extensions"},
	"workspace.info":       {},
	"workspace.session":    {},
	"workspace.extensions": {},
	"container":            {"container.cpu", "container.memory", "container.fs", "container.fd", "container.pids"},
	"container.cpu":        {},
	"container.memory":     {},
	"container.fs":         {},
	"container.fd":         {},
	"container.pids":       {},
	"pressure":             {"pressure.cpu", "pressure.memory", "pressure.io"},
	"pressure.cpu":         {},
	"pressure.memory":      {},
	"pressure.io":          {},
	"network":              {},
	"io":                   {},
	"sockets":              {},
	"gpu":                  {},
}

var allLeafCollectors = []string{
	"container.cpu",
	"container.fd",
	"container.fs",
	"container.memory",
	"container.pids",
	"io",
	"network",
	"pressure.cpu",
	"pressure.io",
	"pressure.memory",
	"sockets",
	"workspace.extensions",
	"workspace.info",
	"workspace.session",
}

func IsCollectorEnabled(name string, collectors []string) bool {
	if len(collectors) == 0 || slices.Contains(collectors, "*") {
		return true
	}

	for _, c := range collectors {
		if c == name || strings.HasPrefix(name, c+".") || strings.HasPrefix(c, name+".") {
			return true
		}
	}

	return false
}

func ExpandCollectors(collectors []string) []string {
	if len(collectors) == 0 || slices.Contains(collectors, "*") {
		result := make([]string, 0, len(allLeafCollectors)+2)
		for _, c := range allLeafCollectors {
			if strings.HasPrefix(c, "pressure.") && !IsPressureAvailable() {
				continue
			}
			result = append(result, c)
		}
		if IsGPUAvailable() {
			result = append(result, "gpu")
		}
		return result
	}

	expanded := make(map[string]bool)
	for _, c := range collectors {
		subs := ValidCollectors[c]
		if len(subs) == 0 {
			expanded[c] = true
			continue
		}
		for _, sub := range subs {
			expanded[sub] = true
		}
	}

	result := make([]string, 0, len(expanded))
	for c := range expanded {
		result = append(result, c)
	}
	slices.Sort(result)
	return result
}
