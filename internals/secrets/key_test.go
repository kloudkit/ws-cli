package secrets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kloudkit/ws-cli/internals/config"
	"gotest.tools/v3/assert"
)

func TestResolveMasterKey(t *testing.T) {
	t.Run("FromFlag", func(t *testing.T) {
		key := "this-is-not-base64-because-of-symbols!"
		resolved, err := ResolveMasterKey(key)

		assert.NilError(t, err)
		assert.Equal(t, key, string(resolved))
	})

	t.Run("FromBase64Flag", func(t *testing.T) {
		rawKey := []byte("12345678901234567890123456789012")

		resolved, err := ResolveMasterKey("MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
		assert.NilError(t, err)
		assert.DeepEqual(t, rawKey, resolved)
	})

	t.Run("FromFile", func(t *testing.T) {
		keyFile := filepath.Join(t.TempDir(), "master.key")
		keyContent := "secretkey"
		err := os.WriteFile(keyFile, []byte(keyContent), 0600)
		assert.NilError(t, err)

		resolved, err := ResolveMasterKey(keyFile)
		assert.NilError(t, err)
		assert.Equal(t, keyContent, string(resolved))
	})

	t.Run("FromEnv", func(t *testing.T) {
		keyContent := "env-secret-key"
		t.Setenv(config.EnvSecretsKey, keyContent)

		resolved, err := ResolveMasterKey("")
		assert.NilError(t, err)
		assert.Equal(t, keyContent, string(resolved))
	})

	t.Run("FromEnvWithPath", func(t *testing.T) {
		keyFile := filepath.Join(t.TempDir(), "master.key")
		err := os.WriteFile(keyFile, []byte("secretkey"), 0600)
		assert.NilError(t, err)

		t.Setenv(config.EnvSecretsKey, keyFile)

		resolved, err := ResolveMasterKey("")
		assert.NilError(t, err)
		assert.Equal(t, keyFile, string(resolved))
	})

	t.Run("FromEnvFile", func(t *testing.T) {
		keyFile := filepath.Join(t.TempDir(), "env.master.key")
		keyContent := "env-file-secret-key"
		err := os.WriteFile(keyFile, []byte(keyContent), 0600)
		assert.NilError(t, err)

		t.Setenv(config.EnvSecretsKeyFile, keyFile)

		resolved, err := ResolveMasterKey("")
		assert.NilError(t, err)
		assert.Equal(t, keyContent, string(resolved))
	})

	t.Run("Precedence", func(t *testing.T) {
		t.Setenv(config.EnvSecretsKey, "env-key")

		resolved, err := ResolveMasterKey("flag-key")
		assert.NilError(t, err)
		assert.Equal(t, "flag-key", string(resolved))
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Setenv(config.EnvSecretsKey, "")
		t.Setenv(config.EnvSecretsKeyFile, "")

		if _, err := os.Stat(config.DefaultSecretsKeyPath); err == nil {
			t.Skip("Skipping test because " + config.DefaultSecretsKeyPath + " exists")
		}

		_, err := ResolveMasterKey("")
		assert.ErrorContains(t, err, "master key not found")
		assert.ErrorContains(t, err, config.DefaultSecretsKeyPath)
	})
}
