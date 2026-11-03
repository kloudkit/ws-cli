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

func setupSecrets(t *testing.T, env string) string {
	t.Helper()
	reference := filepath.Join(t.TempDir(), "env.reference.yaml")
	assert.NilError(t, os.WriteFile(reference, []byte(masterKeyFixture), 0o644))
	t.Setenv("WS__INTERNAL_ENV_REFERENCE", reference)

	root := t.TempDir()
	t.Setenv("WS__INTERNAL_SECRETS_ROOT", root)
	t.Setenv("WS_SECRETS_MASTER_KEY", env)

	return root
}

func writeFile(t *testing.T, path, content string) string {
	t.Helper()
	assert.NilError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NilError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestResolveMasterKey(t *testing.T) {
	keyFile := writeFile(t, filepath.Join(t.TempDir(), "master.key"), "secretkey")
	prefixFile := writeFile(t, filepath.Join(t.TempDir(), "mk"), "file-prefix-key\n")

	cases := []struct {
		name, flag, env, convention, want string
	}{
		{name: "Flag", flag: "this-is-not-base64-because-of-symbols!", want: "this-is-not-base64-because-of-symbols!"},
		{name: "Base64Flag", flag: "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=", want: "12345678901234567890123456789012"},
		{name: "FlagFile", flag: keyFile, want: "secretkey"},
		{name: "Env", env: "env-secret-key", want: "env-secret-key"},
		{name: "FlagWinsOverEnv", flag: "flag-key", env: "env-key", want: "flag-key"},
		{name: "EnvFilePrefix", env: "file:" + prefixFile, want: "file-prefix-key"},
		{name: "Convention", convention: "convention-key\n", want: "convention-key"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := setupSecrets(t, c.env)
			if c.convention != "" {
				writeFile(t, filepath.Join(root, "secrets", "master_key"), c.convention)
			}

			resolved, err := ResolveMasterKey(c.flag)
			assert.NilError(t, err)
			assert.Equal(t, string(resolved), c.want)
		})
	}

	t.Run("NotFound", func(t *testing.T) {
		setupSecrets(t, "")

		_, err := ResolveMasterKey("")
		assert.ErrorContains(t, err, "master key not found")
		assert.ErrorContains(t, err, "/run/secrets/workspace/secrets/master_key")
		assert.ErrorContains(t, err, "file:")
	})
}

func TestMaterializeMasterKey(t *testing.T) {
	source := writeFile(t, filepath.Join(t.TempDir(), "mk"), "file-key\n")

	cases := []struct {
		name, env, existing, want string
		written                   bool
	}{
		{name: "EnvValue", env: "raw-key-value", want: "raw-key-value", written: true},
		{name: "FilePrefix", env: "file:" + source, want: "file-key", written: true},
		{name: "UnsetIsNoop"},
		{name: "ExistingIsNoop", env: "new-value", existing: "original", want: "original"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			target := filepath.Join(setupSecrets(t, c.env), "secrets", "master_key")
			if c.existing != "" {
				writeFile(t, target, c.existing)
			}

			path, err := MaterializeMasterKey()
			assert.NilError(t, err)

			if c.written {
				assert.Equal(t, path, target)
				info, err := os.Stat(path)
				assert.NilError(t, err)
				assert.Equal(t, info.Mode().Perm(), os.FileMode(0o600))
			} else {
				assert.Equal(t, path, "")
			}

			data, _ := os.ReadFile(target)
			assert.Equal(t, string(data), c.want)
		})
	}
}
