package metrics

import (
	"slices"
	"strings"
)

const Namespace = "workspace"

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
