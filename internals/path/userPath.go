package path

import (
	"os"
	"strings"
)

func GetHomeDirectory(segments ...string) string {
	home, exists := os.LookupEnv("HOME")

	if !exists {
		home = "/home/kloud"
	}

  if len(segments) != 0 {
    append := strings.Join(segments, "/")
    append = strings.ReplaceAll(append, "//", "/")
    append = strings.TrimPrefix(append, "/")
    append = strings.TrimSuffix(append, "/")

    home += "/" + append
  }

	return home
}
