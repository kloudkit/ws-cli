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

func GetAll() map[string]string {
	envVars := os.Environ()
	result := make(map[string]string, len(envVars))

	for _, env := range envVars {
		if key, value, ok := strings.Cut(env, "="); ok {
			result[key] = value
		}
	}

	return result
}

func Home() string {
	return String("HOME", "/home/kloud")
}

func IsSSHSession() bool {
	for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		if String(key) != "" {
			return true
		}
	}

	return false
}
