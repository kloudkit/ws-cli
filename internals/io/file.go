package io

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/term"
)

const DefaultFileMode fs.FileMode = 0o600

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func CanOverride(path string, force bool) bool {
	return force || !FileExists(path)
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

	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}

	if err := os.WriteFile(dest, data, 0o666); err != nil {
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

	parentDir := filepath.Dir(filePath)
	dirMode := fileMode&0o077 | 0o700

	if err := os.MkdirAll(parentDir, dirMode); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	if err := os.WriteFile(filePath, content, fileMode); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return os.Chmod(filePath, fileMode)
}

func ReadInput(arg string, reader io.Reader) (string, error) {
	if arg != "-" {
		return arg, nil
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read from stdin: %w", err)
	}

	return string(data), nil
}

func ReadPasswordFromReader(reader io.Reader) (string, error) {
	if file, ok := reader.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		fmt.Fprint(os.Stderr, "Enter password: ")
		password, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(os.Stderr)

		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}

		return string(password), nil
	}

	password, err := ReadInput("-", reader)
	if err == nil && strings.TrimSpace(password) == "" {
		err = errors.New("empty password provided")
	}

	return password, err
}
