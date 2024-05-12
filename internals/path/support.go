package path

import (
	"os"
	"regexp"
	"strings"
)

func AppendSegments(root string, segments ...string) string {
    if len(segments) != 0 {
      root += "/" + strings.Join(segments, "/")
    }

    re := regexp.MustCompile(`/+`)
	  root = re.ReplaceAllString(root, "/")

    return  strings.TrimSuffix(root, "/")
}

func GetHomeDirectory(segments ...string) string {
	home, exists := os.LookupEnv("HOME")

	if !exists {
		home = "/home/kloud"
	}

	return AppendSegments(home, segments...)
}

func GetIPCSocket() string {
	socket, exists := os.LookupEnv("WS_IPC_SOCKET")

	if !exists {
		socket = "/var/workspace/ipc.socket"
	}

	return socket
}

func CanOverride(path_ string, force_ bool) bool {
  if _, err := os.Stat(path_); os.IsNotExist(err) && !force_ {
    return false
  }

  return true
}
