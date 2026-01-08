package path

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/kloudkit/ws-cli/internals/config"
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
	return env.String(config.EnvIPCSocket, config.DefaultIPCSocket)
}

func ResolveConfigPath(configPath string) string {
	if strings.HasPrefix(configPath, "/") {
		return configPath
	}

	return GetHomeDirectory(configPath)
}

func GetCurrentWorkingDirectory(segments ...string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	return AppendSegments(cwd, segments...), nil
}

func ShortenHomePath(path_ string) string {
	homeDir := GetHomeDirectory()

	if after, ok := strings.CutPrefix(path_, homeDir); ok {
		return "~" + after
	}

	return path_
}
