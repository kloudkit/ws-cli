package secrets

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

const masterKeyFixture = `
envs:
  secrets:
    properties:
      master_key:
        type: string
        default: null
        secret: true
`

func _installMasterKeyFixture(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "env.reference.yaml")
	assert.NilError(t, os.WriteFile(path, []byte(masterKeyFixture), 0o644))
	t.Setenv("WS__INTERNAL_ENV_REFERENCE", path)
}

func _newSecretRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("WS__INTERNAL_SECRETS_ROOT", root)
	return root
}

func TestResolveMasterKey(t *testing.T) {
	t.Run("FromFlag", func(t *testing.T) {
		_installMasterKeyFixture(t)
		_newSecretRoot(t)
		key := "this-is-not-base64-because-of-symbols!"
		resolved, err := ResolveMasterKey(key)

		assert.NilError(t, err)
		assert.Equal(t, key, string(resolved))
	})

	t.Run("FromBase64Flag", func(t *testing.T) {
		_installMasterKeyFixture(t)
		_newSecretRoot(t)
		rawKey := []byte("12345678901234567890123456789012")

		resolved, err := ResolveMasterKey("MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
		assert.NilError(t, err)
		assert.DeepEqual(t, rawKey, resolved)
	})

	t.Run("FromFile", func(t *testing.T) {
		_installMasterKeyFixture(t)
		_newSecretRoot(t)
		keyFile := filepath.Join(t.TempDir(), "master.key")
		keyContent := "secretkey"
		err := os.WriteFile(keyFile, []byte(keyContent), 0600)
		assert.NilError(t, err)

		resolved, err := ResolveMasterKey(keyFile)
		assert.NilError(t, err)
		assert.Equal(t, keyContent, string(resolved))
	})

	t.Run("FromEnv", func(t *testing.T) {
		_installMasterKeyFixture(t)
		_newSecretRoot(t)
		keyContent := "env-secret-key"
		t.Setenv("WS_SECRETS_MASTER_KEY", keyContent)

		resolved, err := ResolveMasterKey("")
		assert.NilError(t, err)
		assert.Equal(t, keyContent, string(resolved))
	})

	t.Run("Precedence", func(t *testing.T) {
		_installMasterKeyFixture(t)
		_newSecretRoot(t)
		t.Setenv("WS_SECRETS_MASTER_KEY", "env-key")

		resolved, err := ResolveMasterKey("flag-key")
		assert.NilError(t, err)
		assert.Equal(t, "flag-key", string(resolved))
	})

	t.Run("NotFound", func(t *testing.T) {
		_installMasterKeyFixture(t)
		_newSecretRoot(t)
		t.Setenv("WS_SECRETS_MASTER_KEY", "")

		_, err := ResolveMasterKey("")
		assert.ErrorContains(t, err, "master key not found")
		assert.ErrorContains(t, err, "/run/secrets/workspace/secrets/master_key")
		assert.ErrorContains(t, err, "file:")
	})
}

func TestResolveMasterKey_FromConventionPath(t *testing.T) {
	_installMasterKeyFixture(t)
	root := _newSecretRoot(t)
	keyContent := "convention-key"
	keyPath := filepath.Join(root, "secrets/master_key")
	assert.NilError(t, os.MkdirAll(filepath.Dir(keyPath), 0o755))
	assert.NilError(t, os.WriteFile(keyPath, []byte(keyContent+"\n"), 0o600))
	t.Setenv("WS_SECRETS_MASTER_KEY", "")

	resolved, err := ResolveMasterKey("")
	assert.NilError(t, err)
	assert.Equal(t, keyContent, string(resolved))
}

func TestResolveMasterKey_FromFilePrefix(t *testing.T) {
	_installMasterKeyFixture(t)
	_newSecretRoot(t)
	keyFile := filepath.Join(t.TempDir(), "mk")
	assert.NilError(t, os.WriteFile(keyFile, []byte("file-prefix-key\n"), 0o600))
	t.Setenv("WS_SECRETS_MASTER_KEY", "file:"+keyFile)

	resolved, err := ResolveMasterKey("")
	assert.NilError(t, err)
	assert.Equal(t, "file-prefix-key", string(resolved))
}

func TestMaterializeMasterKey(t *testing.T) {
	t.Run("FromEnvValue", func(t *testing.T) {
		root := _newSecretRoot(t)
		t.Setenv("WS_SECRETS_MASTER_KEY", "raw-key-value")

		path, err := MaterializeMasterKey()
		assert.NilError(t, err)
		assert.Equal(t, path, filepath.Join(root, "secrets", "master_key"))

		data, err := os.ReadFile(path)
		assert.NilError(t, err)
		assert.Equal(t, string(data), "raw-key-value")

		info, err := os.Stat(path)
		assert.NilError(t, err)
		assert.Equal(t, info.Mode().Perm(), os.FileMode(0o600))
	})

	t.Run("FromFilePrefix", func(t *testing.T) {
		_newSecretRoot(t)
		source := filepath.Join(t.TempDir(), "mk")
		assert.NilError(t, os.WriteFile(source, []byte("file-key\n"), 0o600))
		t.Setenv("WS_SECRETS_MASTER_KEY", "file:"+source)

		path, err := MaterializeMasterKey()
		assert.NilError(t, err)

		data, err := os.ReadFile(path)
		assert.NilError(t, err)
		assert.Equal(t, string(data), "file-key")
	})

	t.Run("UnsetIsNoop", func(t *testing.T) {
		_newSecretRoot(t)
		t.Setenv("WS_SECRETS_MASTER_KEY", "")

		path, err := MaterializeMasterKey()
		assert.NilError(t, err)
		assert.Equal(t, path, "")
	})

	t.Run("ExistingIsNoop", func(t *testing.T) {
		root := _newSecretRoot(t)
		existing := filepath.Join(root, "secrets", "master_key")
		assert.NilError(t, os.MkdirAll(filepath.Dir(existing), 0o755))
		assert.NilError(t, os.WriteFile(existing, []byte("original"), 0o600))
		t.Setenv("WS_SECRETS_MASTER_KEY", "new-value")

		path, err := MaterializeMasterKey()
		assert.NilError(t, err)
		assert.Equal(t, path, "")

		data, _ := os.ReadFile(existing)
		assert.Equal(t, string(data), "original")
	})
}
