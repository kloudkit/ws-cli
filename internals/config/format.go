package config

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	boolTruthy = map[string]bool{"1": true, "true": true, "yes": true, "on": true}
	boolFalsy  = map[string]bool{"0": true, "false": true, "no": true, "off": true}
)

func ParseBool(s string) (bool, error) {
	lower := strings.ToLower(s)
	if boolTruthy[lower] {
		return true, nil
	}
	if boolFalsy[lower] {
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
	parts := strings.Split(s, delim)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
