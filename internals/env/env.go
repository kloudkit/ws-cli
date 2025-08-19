package env

import "os"

func String(key string, fallback ...string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}

	if len(fallback) > 0 {
		return fallback[0]
	}

	return ""
}

func MustString(key string, fallback ...string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}

	if len(fallback) > 0 {
		return fallback[0]
	}

	panic("environment variable " + key + " not set")
}
