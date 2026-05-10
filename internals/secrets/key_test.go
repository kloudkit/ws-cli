package secrets

import (
	"os"
	"path/filepath"
	"testing"

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
		t.Setenv("WS_SECRETS_MASTER_KEY", keyContent)

		resolved, err := ResolveMasterKey("")
		assert.NilError(t, err)
		assert.Equal(t, keyContent, string(resolved))
	})

	t.Run("FromEnvWithPath", func(t *testing.T) {
		keyFile := filepath.Join(t.TempDir(), "master.key")
		err := os.WriteFile(keyFile, []byte("secretkey"), 0600)
		assert.NilError(t, err)

		t.Setenv("WS_SECRETS_MASTER_KEY", keyFile)

		resolved, err := ResolveMasterKey("")
		assert.NilError(t, err)
		assert.Equal(t, keyFile, string(resolved))
	})

	t.Run("FromEnvFile", func(t *testing.T) {
		keyFile := filepath.Join(t.TempDir(), "env.master.key")
		keyContent := "env-file-secret-key"
		err := os.WriteFile(keyFile, []byte(keyContent), 0600)
		assert.NilError(t, err)

		t.Setenv("WS_SECRETS_MASTER_KEY_FILE", keyFile)

		resolved, err := ResolveMasterKey("")
		assert.NilError(t, err)
		assert.Equal(t, keyContent, string(resolved))
	})

	t.Run("Precedence", func(t *testing.T) {
		t.Setenv("WS_SECRETS_MASTER_KEY", "env-key")

		resolved, err := ResolveMasterKey("flag-key")
		assert.NilError(t, err)
		assert.Equal(t, "flag-key", string(resolved))
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Setenv("WS_SECRETS_MASTER_KEY", "")
		t.Setenv("WS_SECRETS_MASTER_KEY_FILE", "")

		if _, err := os.Stat("/etc/workspace/master.key"); err == nil {
			t.Skip("Skipping test because " + "/etc/workspace/master.key" + " exists")
		}

		_, err := ResolveMasterKey("")
		assert.ErrorContains(t, err, "master key not found")
		assert.ErrorContains(t, err, "/etc/workspace/master.key")
	})
}
