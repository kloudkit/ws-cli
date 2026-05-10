package config

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	}
	return false, fmt.Errorf("not a boolean: %q (accepted: 1/true/yes/on or 0/false/no/off)", s)
}

func ParseInt(s string) (int64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("not an integer: %q", s)
	}
	return n, nil
}

func ParseList(s, delim string) []string {
	if s == "" {
		return nil
	}
	if delim == "" {
		delim = " "
	}
	out := []string{}
	for _, p := range strings.Split(s, delim) {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
