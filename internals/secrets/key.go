package secrets

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/kloudkit/ws-cli/internals/io"
)

func ResolveMasterKey(flagValue string) ([]byte, error) {
	if flagValue != "" {
		if io.FileExists(flagValue) {
			return readKeyFile(flagValue)
		}

		return parseKey(flagValue)
	}

	if val := env.String(config.EnvSecretsKey); val != "" {
		return parseKey(val)
	}

	if filePath := env.String(config.EnvSecretsKeyFile); filePath != "" {
		if !io.FileExists(filePath) {
			return nil, fmt.Errorf("master key file not found at %s: %s", config.EnvSecretsKeyFile, filePath)
		}

		return readKeyFile(filePath)
	}

	if io.FileExists(config.DefaultSecretsKeyPath) {
		return readKeyFile(config.DefaultSecretsKeyPath)
	}

	return nil, fmt.Errorf(
		"master key not found (use --master, %s, %s, or check %s)",
		config.EnvSecretsKey,
		config.EnvSecretsKeyFile,
		config.DefaultSecretsKeyPath,
	)
}

func readKeyFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)

	if err != nil {
		return nil, fmt.Errorf("failed to read master key file: %w", err)
	}

	return parseKey(string(data))
}

func parseKey(keyStr string) ([]byte, error) {
	keyStr = strings.TrimSpace(keyStr)

	if decoded, err := base64.StdEncoding.DecodeString(keyStr); err == nil && len(decoded) >= 16 {
		return decoded, nil
	}

	return []byte(keyStr), nil
}
