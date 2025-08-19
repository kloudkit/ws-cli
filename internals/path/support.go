package path

import (
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
