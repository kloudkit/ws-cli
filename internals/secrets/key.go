package secrets

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/io"
)

func ResolveMasterKey(flagValue string) ([]byte, error) {
	if flagValue != "" {
		if io.FileExists(flagValue) {
			return readKeyFile(flagValue)
		}

		return parseKey(flagValue)
	}

	if val, _ := config.Resolve("secrets", "master_key"); val != "" {
		return parseKey(val)
	}

	return nil, fmt.Errorf(
		"master key not found (use --master, WS_SECRETS_MASTER_KEY=<value>, " +
			"WS_SECRETS_MASTER_KEY=file:/path, or mount the key at " +
			"/run/secrets/workspace/secrets/master_key)",
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
