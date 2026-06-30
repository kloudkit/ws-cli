package seed

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const tempSuffix = ".ws-seed.tmp"

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

	dirMode := mode&0o077 | 0o700
	if relDir := filepath.Dir(rel); relDir != "." {
		if err := root.MkdirAll(relDir, dirMode); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}
	}

	tmp := rel + tempSuffix
	if err := writeTemp(root, tmp, content, mode); err != nil {
		return err
	}

	if err := root.Rename(tmp, rel); err != nil {
		root.Remove(tmp)
		return fmt.Errorf("failed to rename into place: %w", err)
	}

	return nil
}

func writeTemp(root *os.Root, tmp string, content []byte, mode fs.FileMode) (err error) {
	file, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if errors.Is(err, fs.ErrExist) {
		root.Remove(tmp)
		file, err = root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	}
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close temp file: %w", cerr)
		}
		if err != nil {
			root.Remove(tmp)
		}
	}()

	if _, err = file.Write(content); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err = file.Chmod(mode); err != nil {
		return fmt.Errorf("failed to set mode: %w", err)
	}

	return nil
}
