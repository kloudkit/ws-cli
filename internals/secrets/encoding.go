package secrets

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const base64Prefix = "base64:"

func EncodeWithPrefix(data []byte) string {
	return base64Prefix + base64.StdEncoding.EncodeToString(data)
}

func DecodeWithPrefix(encoded string) ([]byte, error) {
	if !strings.HasPrefix(encoded, base64Prefix) {
		return []byte(encoded), nil
	}

	trimmed := strings.TrimPrefix(encoded, base64Prefix)
	decoded, err := base64.StdEncoding.DecodeString(trimmed)

	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 input: %w", err)
	}

	return decoded, nil
}
