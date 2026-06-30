package seed

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

var consumedDirs = []string{
	".ws/startup.d",
	".ws/ca.d",
	".ws/session.d",
	".ws/features.d",
}

func nearestExistingAncestor(dest string) string {
	current := filepath.Dir(dest)

	for {
		if _, err := os.Lstat(current); err == nil {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			return current
		}

		current = parent
	}
}

func ownsPath(p string) bool {
	info, err := os.Stat(p)
	if err != nil {
		return false
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}

	return int(stat.Uid) == os.Geteuid()
}

func isUnder(p, root string) bool {
	if root == "" {
		return false
	}

	root = filepath.Clean(root)
	p = filepath.Clean(p)

	return p == root || strings.HasPrefix(p, root+string(os.PathSeparator))
}

func chooseAnchor(dest string, vars Vars, ancestor string) string {
	if isUnder(dest, vars.Home) {
		return vars.Home
	}

	if isUnder(dest, vars.ServerRoot) {
		return vars.ServerRoot
	}

	return ancestor
}

func consumedNotice(dest, home string) bool {
	for _, dir := range consumedDirs {
		if isUnder(dest, filepath.Join(home, dir)) {
			return true
		}
	}

	return false
}
