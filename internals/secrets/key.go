package secrets

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/io"
)

func MaterializeMasterKey() (string, error) {
	value := os.Getenv(config.RuntimeKey("secrets", "master_key"))
	path := config.SecretConventionPath("secrets", "master_key")

	if value == "" || io.FileExists(path) {
		return "", nil
	}

	key := value

	if source, found := strings.CutPrefix(value, "file:"); found {
		data, err := os.ReadFile(source)
		if err != nil {
			return "", nil
		}

		key = strings.TrimRight(string(data), "\n")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := file.Write([]byte(key)); err != nil {
		return "", err
	}

	return path, nil
}

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
