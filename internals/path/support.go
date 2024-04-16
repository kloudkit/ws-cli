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
    // root = strings.TrimPrefix(root, "/")

    return  strings.TrimSuffix(root, "/")
}

func GetHomeDirectory(segments ...string) string {
	home, exists := os.LookupEnv("HOME")

	if !exists {
		home = "/home/kloud"
	}

	return AppendSegments(home, segments...)
}
