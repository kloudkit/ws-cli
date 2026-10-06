package info

import "runtime/debug"

func Version() string {
	if build, ok := debug.ReadBuildInfo(); ok && build.Main.Version != "" {
		return build.Main.Version
	}

	return "(devel)"
}
