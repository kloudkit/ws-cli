package seed

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

const tempSuffix = ".ws-seed.tmp"

var consumedDirs = []string{".ws/startup.d", ".ws/ca.d", ".ws/session.d", ".ws/features.d"}

func nearestExistingAncestor(dest string) string {
	for current := filepath.Dir(dest); ; current = filepath.Dir(current) {
		if _, err := os.Lstat(current); err == nil || current == filepath.Dir(current) {
			return current
		}
	}
}

func ownsPath(p string) bool {
	info, err := os.Stat(p)
	if err != nil {
		return false
	}

	stat, ok := info.Sys().(*syscall.Stat_t)

	return ok && int(stat.Uid) == os.Geteuid()
}

func isUnder(p, root string) bool {
	if root == "" {
		return false
	}

	root, p = filepath.Clean(root), filepath.Clean(p)

	return p == root || strings.HasPrefix(p, strings.TrimSuffix(root, string(os.PathSeparator))+string(os.PathSeparator))
}

func chooseAnchor(dest string, vars Vars, ancestor string) string {
	for _, root := range []string{vars.Home, vars.ServerRoot} {
		if isUnder(dest, root) {
			return root
		}
	}

	return ancestor
}

func consumedNotice(dest, home string) bool {
	return slices.ContainsFunc(consumedDirs, func(dir string) bool {
		return isUnder(dest, filepath.Join(home, dir))
	})
}

func writeAtomic(anchor, dest string, content []byte, mode fs.FileMode) error {
	root, err := os.OpenRoot(anchor)
	if err != nil {
		return fmt.Errorf("failed to open root %q: %w", anchor, err)
	}
	defer root.Close()

	rel, err := filepath.Rel(anchor, dest)
	if err != nil {
		return fmt.Errorf("failed to resolve relative path: %w", err)
	}

	if info, err := root.Lstat(rel); err == nil && info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("refusing to write through symlink %q", dest)
	}

	if relDir := filepath.Dir(rel); relDir != "." {
		if err := root.MkdirAll(relDir, mode&0o077|0o700); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}
	}

	tmp := rel + tempSuffix
	root.Remove(tmp)

	file, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	step := "write temp file"
	if _, err = file.Write(content); err == nil {
		step, err = "set mode", file.Chmod(mode)
	}

	if closeErr := file.Close(); err == nil && closeErr != nil {
		step, err = "close temp file", closeErr
	}

	if err == nil {
		step, err = "rename into place", root.Rename(tmp, rel)
	}

	if err != nil {
		root.Remove(tmp)
		return fmt.Errorf("failed to %s: %w", step, err)
	}

	return nil
}
