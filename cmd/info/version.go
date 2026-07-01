package info

import "runtime/debug"

func Version() string {
	if build, ok := debug.ReadBuildInfo(); ok {
		return resolveVersion(build.Main.Version)
	}

	return resolveVersion("")
}

func resolveVersion(v string) string {
	if v == "" || v == "(devel)" {
		return "(devel)"
	}

	return v
}
