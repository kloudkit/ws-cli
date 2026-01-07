package secrets

import (
	"os"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestEncryptDecrypt(t *testing.T) {
	masterKey := make([]byte, 32)
	plainText := "secret data"

	encrypted, err := Encrypt([]byte(plainText), masterKey)
	assert.NilError(t, err)
	assert.Assert(t, strings.HasPrefix(encrypted, "argon2id$"))

	decrypted, err := Decrypt(encrypted, masterKey)
	assert.NilError(t, err)
	assert.Equal(t, plainText, string(decrypted))
}

func TestDecryptInvalidFormat(t *testing.T) {
	masterKey := make([]byte, 32)
	_, err := Decrypt("invalid", masterKey)

	assert.ErrorContains(t, err, "invalid encoded format")
}

func TestDecryptUnsupportedAlgorithm(t *testing.T) {
	masterKey := make([]byte, 32)
	encoded := "sha256$v=1$m=1,t=1,p=1$salt$cipher"
	_, err := Decrypt(encoded, masterKey)

	assert.ErrorContains(t, err, "unsupported algorithm")
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := []byte("12345678901234567890123456789012")
	key2 := []byte("22345678901234567890123456789012")
	plainText := "data"

	encrypted, err := Encrypt([]byte(plainText), key1)
	assert.NilError(t, err)

	_, err = Decrypt(encrypted, key2)
	assert.ErrorContains(t, err, "message authentication failed")
}

func TestLoadVaultFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	vaultFile := tmpDir + "/vault.yaml"

	vaultContent := `secrets:
  - type: kubeconfig
    value: encrypted_value_1
    destination: /home/dev/.kube/config
    force: true
  - type: env
    value: encrypted_value_2
    destination: MY_SECRET
`

	err := os.WriteFile(vaultFile, []byte(vaultContent), 0644)
	assert.NilError(t, err)

	vault, err := LoadVaultFromFile(vaultFile)
	assert.NilError(t, err)
	assert.Equal(t, 2, len(vault.Secrets))
	assert.Equal(t, "kubeconfig", vault.Secrets[0].Type)
	assert.Equal(t, "encrypted_value_1", vault.Secrets[0].Value)
	assert.Equal(t, "/home/dev/.kube/config", vault.Secrets[0].Destination)
	assert.Equal(t, true, vault.Secrets[0].Force)
	assert.Equal(t, "env", vault.Secrets[1].Type)
	assert.Equal(t, "MY_SECRET", vault.Secrets[1].Destination)
}

func TestLoadVaultFromFileNotFound(t *testing.T) {
	_, err := LoadVaultFromFile("/nonexistent/vault.yaml")
	assert.ErrorContains(t, err, "failed to read vault file")
}

func TestLoadVaultFromFileInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	vaultFile := tmpDir + "/invalid.yaml"

	err := os.WriteFile(vaultFile, []byte("invalid: yaml: content: ["), 0644)
	assert.NilError(t, err)

	_, err = LoadVaultFromFile(vaultFile)
	assert.ErrorContains(t, err, "failed to parse vault YAML")
}

func TestVaultEncryptAll(t *testing.T) {
	tmpDir := t.TempDir()

	secretFile1 := tmpDir + "/.kube/config"
	err := os.MkdirAll(tmpDir+"/.kube", 0755)
	assert.NilError(t, err)
	err = os.WriteFile(secretFile1, []byte("kubeconfig content"), 0600)
	assert.NilError(t, err)

	os.Setenv("MY_ENV_SECRET", "env secret value")
	defer os.Unsetenv("MY_ENV_SECRET")

	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	vault := &Vault{
		Secrets: []Secret{
			{
				Type:        "kubeconfig",
				Destination: secretFile1,
			},
			{
				Type:        "env",
				Destination: "MY_ENV_SECRET",
			},
		},
	}

	masterKey := make([]byte, 32)
	err = vault.EncryptAll(masterKey)
	assert.NilError(t, err)

	assert.Assert(t, strings.HasPrefix(vault.Secrets[0].Value, "argon2id$"))
	assert.Assert(t, strings.HasPrefix(vault.Secrets[1].Value, "argon2id$"))

	decrypted1, err := Decrypt(vault.Secrets[0].Value, masterKey)
	assert.NilError(t, err)
	assert.Equal(t, "kubeconfig content", string(decrypted1))

	decrypted2, err := Decrypt(vault.Secrets[1].Value, masterKey)
	assert.NilError(t, err)
	assert.Equal(t, "env secret value", string(decrypted2))
}

func TestVaultEncryptAllFileNotFound(t *testing.T) {
	vault := &Vault{
		Secrets: []Secret{
			{
				Type:        "kubeconfig",
				Destination: "/nonexistent/file",
			},
		},
	}

	masterKey := make([]byte, 32)
	err := vault.EncryptAll(masterKey)
	assert.ErrorContains(t, err, "failed to read secret value")
}

func TestVaultDecryptAll(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := tmpDir + "/.kube/config"

	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	masterKey := make([]byte, 32)
	encrypted, err := Encrypt([]byte("kubeconfig content"), masterKey)
	assert.NilError(t, err)

	vault := &Vault{
		Secrets: []Secret{
			{
				Type:        "kubeconfig",
				Value:       encrypted,
				Destination: outputFile,
				Force:       false,
			},
		},
	}

	opts := WriteOptions{Force: false, DryRun: false}
	err = vault.DecryptAll(masterKey, opts)
	assert.NilError(t, err)

	written, err := os.ReadFile(outputFile)
	assert.NilError(t, err)
	assert.Equal(t, "kubeconfig content", string(written))
}

func TestVaultDecryptAllDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := tmpDir + "/.kube/config"

	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	masterKey := make([]byte, 32)
	encrypted, err := Encrypt([]byte("content"), masterKey)
	assert.NilError(t, err)

	vault := &Vault{
		Secrets: []Secret{
			{
				Type:        "kubeconfig",
				Value:       encrypted,
				Destination: outputFile,
			},
		},
	}

	opts := WriteOptions{Force: false, DryRun: true}
	err = vault.DecryptAll(masterKey, opts)
	assert.NilError(t, err)

	_, err = os.Stat(outputFile)
	assert.Assert(t, os.IsNotExist(err))
}

func TestVaultDecryptAllForceOverride(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := tmpDir + "/.kube/config"

	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	err := os.MkdirAll(tmpDir+"/.kube", 0755)
	assert.NilError(t, err)
	err = os.WriteFile(outputFile, []byte("existing"), 0644)
	assert.NilError(t, err)

	masterKey := make([]byte, 32)
	encrypted, err := Encrypt([]byte("new content"), masterKey)
	assert.NilError(t, err)

	vault := &Vault{
		Secrets: []Secret{
			{
				Type:        "kubeconfig",
				Value:       encrypted,
				Destination: outputFile,
				Force:       false,
			},
		},
	}

	opts := WriteOptions{Force: true, DryRun: false}
	err = vault.DecryptAll(masterKey, opts)
	assert.NilError(t, err)

	written, err := os.ReadFile(outputFile)
	assert.NilError(t, err)
	assert.Equal(t, "new content", string(written))
}

func TestVaultDecryptAllSecretForceFlag(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := tmpDir + "/.kube/config"

	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	err := os.MkdirAll(tmpDir+"/.kube", 0755)
	assert.NilError(t, err)
	err = os.WriteFile(outputFile, []byte("existing"), 0644)
	assert.NilError(t, err)

	masterKey := make([]byte, 32)
	encrypted, err := Encrypt([]byte("new content"), masterKey)
	assert.NilError(t, err)

	vault := &Vault{
		Secrets: []Secret{
			{
				Type:        "kubeconfig",
				Value:       encrypted,
				Destination: outputFile,
				Force:       true,
			},
		},
	}

	opts := WriteOptions{Force: false, DryRun: false}
	err = vault.DecryptAll(masterKey, opts)
	assert.NilError(t, err)

	written, err := os.ReadFile(outputFile)
	assert.NilError(t, err)
	assert.Equal(t, "new content", string(written))
}

func TestVaultDecryptAllEmptyValue(t *testing.T) {
	vault := &Vault{
		Secrets: []Secret{
			{
				Type:        "kubeconfig",
				Value:       "",
				Destination: "/home/dev/.kube/config",
			},
		},
	}

	masterKey := make([]byte, 32)
	opts := WriteOptions{Force: false, DryRun: false}
	err := vault.DecryptAll(masterKey, opts)
	assert.ErrorContains(t, err, "has empty value")
}

func TestVaultToYAML(t *testing.T) {
	vault := &Vault{
		Secrets: []Secret{
			{
				Type:        "kubeconfig",
				Value:       "encrypted_value",
				Destination: "/home/dev/.kube/config",
				Force:       true,
			},
			{
				Type:        "env",
				Value:       "encrypted_env",
				Destination: "MY_VAR",
			},
		},
	}

	yamlData, err := vault.ToYAML()
	assert.NilError(t, err)

	assert.Assert(t, strings.Contains(string(yamlData), "type: kubeconfig"))
	assert.Assert(t, strings.Contains(string(yamlData), "value: encrypted_value"))
	assert.Assert(t, strings.Contains(string(yamlData), "destination: /home/dev/.kube/config"))
	assert.Assert(t, strings.Contains(string(yamlData), "force: true"))
	assert.Assert(t, strings.Contains(string(yamlData), "type: env"))
	assert.Assert(t, strings.Contains(string(yamlData), "destination: MY_VAR"))
}

func TestVaultToYAMLEmpty(t *testing.T) {
	vault := &Vault{
		Secrets: []Secret{},
	}

	yamlData, err := vault.ToYAML()
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(yamlData), "secrets: []"))
}
