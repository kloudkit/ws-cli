package secrets

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestResolveMasterKeyFromFlag(t *testing.T) {
	key := "this-is-not-base64-because-of-symbols!"
	resolved, err := ResolveMasterKey(key)

	assert.NilError(t, err)
	assert.Equal(t, key, string(resolved))
}

func TestResolveMasterKeyFromBase64Flag(t *testing.T) {
	rawKey := []byte("12345678901234567890123456789012")

	resolved, err := ResolveMasterKey("MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	assert.NilError(t, err)
	assert.DeepEqual(t, rawKey, resolved)
}

func TestResolveMasterKeyFromFile(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "master.key")
	keyContent := "secretkey"
	err := os.WriteFile(keyFile, []byte(keyContent), 0600)
	assert.NilError(t, err)

	resolved, err := ResolveMasterKey(keyFile)
	assert.NilError(t, err)
	assert.Equal(t, keyContent, string(resolved))
}

func TestResolveMasterKeyFromEnv(t *testing.T) {
	keyContent := "env-secret-key"
	os.Setenv(EnvMasterKey, keyContent)
	defer os.Unsetenv(EnvMasterKey)

	resolved, err := ResolveMasterKey("")
	assert.NilError(t, err)
	assert.Equal(t, keyContent, string(resolved))
}

func TestResolveMasterKeyFromEnvWithPath(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "master.key")
	err := os.WriteFile(keyFile, []byte("secretkey"), 0600)
	assert.NilError(t, err)

	os.Setenv(EnvMasterKey, keyFile)
	defer os.Unsetenv(EnvMasterKey)

	resolved, err := ResolveMasterKey("")
	assert.NilError(t, err)
	assert.Equal(t, keyFile, string(resolved))
}

func TestResolveMasterKeyFromEnvFile(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "env.master.key")
	keyContent := "env-file-secret-key"
	err := os.WriteFile(keyFile, []byte(keyContent), 0600)
	assert.NilError(t, err)

	os.Setenv(EnvMasterKeyFile, keyFile)
	defer os.Unsetenv(EnvMasterKeyFile)

	resolved, err := ResolveMasterKey("")
	assert.NilError(t, err)
	assert.Equal(t, keyContent, string(resolved))
}

func TestResolveMasterKeyPrecedence(t *testing.T) {
	os.Setenv(EnvMasterKey, "env-key")
	defer os.Unsetenv(EnvMasterKey)

	resolved, err := ResolveMasterKey("flag-key")
	assert.NilError(t, err)
	assert.Equal(t, "flag-key", string(resolved))
}

func TestResolveMasterKeyNotFound(t *testing.T) {
	os.Unsetenv(EnvMasterKey)
	os.Unsetenv(EnvMasterKeyFile)

	if _, err := os.Stat(DefaultMasterPath); err == nil {
		t.Skip("Skipping test because " + DefaultMasterPath + " exists")
	}

	_, err := ResolveMasterKey("")
	assert.ErrorContains(t, err, "master key not found")
	assert.ErrorContains(t, err, DefaultMasterPath)
}
