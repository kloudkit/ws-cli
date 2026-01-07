package secrets

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/kloudkit/ws-cli/internals/path"
)

func ResolveMasterKey(flagValue string) ([]byte, error) {
	if flagValue != "" {
		if path.FileExists(flagValue) {
			return readKeyFile(flagValue)
		}

		return parseKey(flagValue)
	}

	if val := env.String(EnvMasterKey); val != "" {
		return parseKey(val)
	}

	if filePath := env.String(EnvMasterKeyFile); filePath != "" {
		if !path.FileExists(filePath) {
			return nil, fmt.Errorf("master key file not found at %s: %s", EnvMasterKeyFile, filePath)
		}

		return readKeyFile(filePath)
	}

	if path.FileExists(DefaultMasterPath) {
		return readKeyFile(DefaultMasterPath)
	}

	return nil, fmt.Errorf(
		"master key not found (use --master, %s, %s, or check %s)",
		EnvMasterKey,
		EnvMasterKeyFile,
		DefaultMasterPath,
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
