package path

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kloudkit/ws-cli/internals/env"
)

var slashRunRe = regexp.MustCompile(`/+`)

func AppendSegments(root string, segments ...string) string {
	if len(segments) != 0 {
		root += "/" + strings.Join(segments, "/")
	}

	root = slashRunRe.ReplaceAllString(root, "/")

	return strings.TrimSuffix(root, "/")
}

func GetHomeDirectory(segments ...string) string {
	return AppendSegments(env.Home(), segments...)
}

func ResolveConfigPath(configPath string) string {
	if strings.HasPrefix(configPath, "/") {
		return configPath
	}

	return GetHomeDirectory(configPath)
}

func ExpandHome(path string) string {
	if path == "~" {
		return env.Home()
	}

	if after, ok := strings.CutPrefix(path, "~/"); ok {
		return env.Home() + "/" + after
	}

	return path
}

func Expand(path string) string {
	return ExpandHome(filepath.Clean(os.ExpandEnv(path)))
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
