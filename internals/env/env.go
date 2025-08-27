package env

import (
	"os"
	"strings"
)

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

func GetAll() map[string]string {
	envVars := os.Environ()
	result := make(map[string]string, len(envVars))

	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)

		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}

	return result
}
