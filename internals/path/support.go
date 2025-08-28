package path

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/kloudkit/ws-cli/internals/env"
)

func AppendSegments(root string, segments ...string) string {
	if len(segments) != 0 {
		root += "/" + strings.Join(segments, "/")
	}

	re := regexp.MustCompile(`/+`)
	root = re.ReplaceAllString(root, "/")

	return strings.TrimSuffix(root, "/")
}

func GetHomeDirectory(segments ...string) string {
	return AppendSegments(env.String("HOME", "/home/kloud"), segments...)
}

func GetIPCSocket() string {
	return env.String("WS_IPC_SOCKET", "/var/workspace/ipc.socket")
}

func CanOverride(path_ string, force bool) bool {
	if _, err := os.Stat(path_); os.IsNotExist(err) || force {
		return true
	}

	return false
}

func ResolveConfigPath(configPath string) string {
	if strings.HasPrefix(configPath, "/") {
		return configPath
	}

	return GetHomeDirectory(configPath)
}

func FileExists(path_ string) bool {
	_, err := os.Stat(path_)

	return !os.IsNotExist(err)
}

func GetCurrentWorkingDirectory(segments ...string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	return AppendSegments(cwd, segments...), nil
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

func ShortenHomePath(path_ string) string {
	homeDir := GetHomeDirectory()

	if after, ok := strings.CutPrefix(path_, homeDir); ok {
		return "~" + after
	}

	return path_
}
