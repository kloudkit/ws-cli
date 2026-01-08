package io

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

const DefaultFileMode fs.FileMode = 0o600

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func CanOverride(path string, force bool) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) || force {
		return true
	}
	return false
}

func ParseFileMode(modeStr string) (fs.FileMode, error) {
	modeStr = strings.TrimSpace(modeStr)
	if modeStr == "" {
		return DefaultFileMode, nil
	}

	var mode uint64
	var err error

	if strings.HasPrefix(modeStr, "0o") || strings.HasPrefix(modeStr, "0O") {
		mode, err = strconv.ParseUint(modeStr[2:], 8, 32)
	} else {
		mode, err = strconv.ParseUint(modeStr, 10, 32)
	}

	if err != nil {
		return 0, fmt.Errorf("invalid file mode '%s': %w", modeStr, err)
	}

	if mode > 0o777 {
		return 0, fmt.Errorf("invalid file mode '%s': exceeds 0o777", modeStr)
	}

	return fs.FileMode(mode), nil
}

func CopyFile(source, dest string) error {
	stats, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	if !stats.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", source)
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

func WriteSecureFile(filePath string, content []byte, modeStr string, force bool) error {
	if !CanOverride(filePath, force) {
		return fmt.Errorf("file %s exists, use --force to overwrite", filePath)
	}

	fileMode, err := ParseFileMode(modeStr)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filePath, content, fileMode); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}
