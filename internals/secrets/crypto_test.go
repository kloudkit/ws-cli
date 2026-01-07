package secrets

import (
	"os"
	"path/filepath"
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

func TestDecryptErrors(t *testing.T) {
	tests := []struct {
		name          string
		encoded       string
		masterKey     []byte
		errorContains string
	}{
		{
			name:          "invalid format",
			encoded:       "invalid",
			masterKey:     make([]byte, 32),
			errorContains: "invalid encoded format",
		},
		{
			name:          "unsupported algorithm",
			encoded:       "sha256$v=1$m=1,t=1,p=1$salt$cipher",
			masterKey:     make([]byte, 32),
			errorContains: "unsupported algorithm",
		},
		{
			name: "wrong key",
			encoded: func() string {
				key1 := []byte("12345678901234567890123456789012")
				enc, _ := Encrypt([]byte("data"), key1)
				return enc
			}(),
			masterKey:     []byte("22345678901234567890123456789012"),
			errorContains: "message authentication failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.encoded, tt.masterKey)
			assert.ErrorContains(t, err, tt.errorContains)
		})
	}
}

func TestResolveMasterKey(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, tmpDir string) string
		cleanup  func()
		expected string
	}{
		{
			name: "plain text flag",
			setup: func(t *testing.T, tmpDir string) string {
				return "this-is-not-base64-because-of-symbols!"
			},
			expected: "this-is-not-base64-because-of-symbols!",
		},
		{
			name: "base64 flag",
			setup: func(t *testing.T, tmpDir string) string {
				return "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI="
			},
			expected: "12345678901234567890123456789012",
		},
		{
			name: "file path",
			setup: func(t *testing.T, tmpDir string) string {
				keyFile := filepath.Join(tmpDir, "master.key")
				os.WriteFile(keyFile, []byte("secretkey"), 0600)
				return keyFile
			},
			expected: "secretkey",
		},
		{
			name: "from env",
			setup: func(t *testing.T, tmpDir string) string {
				os.Setenv(EnvMasterKey, "env-secret-key")
				return ""
			},
			cleanup: func() {
				os.Unsetenv(EnvMasterKey)
			},
			expected: "env-secret-key",
		},
		{
			name: "from env file",
			setup: func(t *testing.T, tmpDir string) string {
				keyFile := filepath.Join(tmpDir, "env.master.key")
				os.WriteFile(keyFile, []byte("env-file-secret-key"), 0600)
				os.Setenv(EnvMasterKeyFile, keyFile)
				return ""
			},
			cleanup: func() {
				os.Unsetenv(EnvMasterKeyFile)
			},
			expected: "env-file-secret-key",
		},
		{
			name: "flag precedence over env",
			setup: func(t *testing.T, tmpDir string) string {
				os.Setenv(EnvMasterKey, "env-key")
				return "flag-key"
			},
			cleanup: func() {
				os.Unsetenv(EnvMasterKey)
			},
			expected: "flag-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			flagValue := tt.setup(t, tmpDir)
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			resolved, err := ResolveMasterKey(flagValue)
			assert.NilError(t, err)
			assert.Equal(t, tt.expected, string(resolved))
		})
	}
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

func TestLoadVaultFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	vaultFile := filepath.Join(tmpDir, "vault.yaml")

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

	expected := &Vault{
		Secrets: []Secret{
			{
				Type:        "kubeconfig",
				Value:       "encrypted_value_1",
				Destination: "/home/dev/.kube/config",
				Force:       true,
			},
			{
				Type:        "env",
				Value:       "encrypted_value_2",
				Destination: "MY_SECRET",
			},
		},
	}

	assert.DeepEqual(t, expected, vault)
}

func TestLoadVaultErrors(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T, tmpDir string) string
		errorContains string
	}{
		{
			name: "file not found",
			setup: func(t *testing.T, tmpDir string) string {
				return "/nonexistent/vault.yaml"
			},
			errorContains: "failed to read vault file",
		},
		{
			name: "invalid yaml",
			setup: func(t *testing.T, tmpDir string) string {
				vaultFile := filepath.Join(tmpDir, "invalid.yaml")
				os.WriteFile(vaultFile, []byte("invalid: yaml: content: ["), 0644)
				return vaultFile
			},
			errorContains: "failed to parse vault YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			vaultFile := tt.setup(t, tmpDir)
			_, err := LoadVaultFromFile(vaultFile)
			assert.ErrorContains(t, err, tt.errorContains)
		})
	}
}

func TestVaultEncryptAll(t *testing.T) {
	tmpDir := t.TempDir()

	secretFile := filepath.Join(tmpDir, ".kube", "config")
	err := os.MkdirAll(filepath.Dir(secretFile), 0755)
	assert.NilError(t, err)
	err = os.WriteFile(secretFile, []byte("kubeconfig content"), 0600)
	assert.NilError(t, err)

	os.Setenv("MY_ENV_SECRET", "env secret value")
	defer os.Unsetenv("MY_ENV_SECRET")

	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	vault := &Vault{
		Secrets: []Secret{
			{Type: "kubeconfig", Destination: secretFile},
			{Type: "env", Destination: "MY_ENV_SECRET"},
		},
	}

	masterKey := make([]byte, 32)
	err = vault.EncryptAll(masterKey)
	assert.NilError(t, err)

	for i, secret := range vault.Secrets {
		assert.Assert(t, strings.HasPrefix(secret.Value, "argon2id$"), "secret %d not encrypted", i)
		decrypted, err := Decrypt(secret.Value, masterKey)
		assert.NilError(t, err)

		if i == 0 {
			assert.Equal(t, "kubeconfig content", string(decrypted))
		} else {
			assert.Equal(t, "env secret value", string(decrypted))
		}
	}
}

func TestVaultEncryptAllFileNotFound(t *testing.T) {
	vault := &Vault{
		Secrets: []Secret{
			{Type: "kubeconfig", Destination: "/nonexistent/file"},
		},
	}

	err := vault.EncryptAll(make([]byte, 32))
	assert.ErrorContains(t, err, "failed to read secret value")
}

func TestVaultDecryptAll(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, ".kube", "config")

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

func TestVaultDecryptAllOptions(t *testing.T) {
	tests := []struct {
		name            string
		existingContent string
		secretForce     bool
		optsForce       bool
		optsDryRun      bool
		expectWrite     bool
		expectError     bool
	}{
		{
			name:        "dry run prevents write",
			secretForce: false,
			optsForce:   false,
			optsDryRun:  true,
			expectWrite: false,
			expectError: false,
		},
		{
			name:            "existing file without force fails",
			existingContent: "existing",
			secretForce:     false,
			optsForce:       false,
			optsDryRun:      false,
			expectWrite:     false,
			expectError:     true,
		},
		{
			name:            "global force overrides",
			existingContent: "existing",
			secretForce:     false,
			optsForce:       true,
			optsDryRun:      false,
			expectWrite:     true,
			expectError:     false,
		},
		{
			name:            "secret force overrides",
			existingContent: "existing",
			secretForce:     true,
			optsForce:       false,
			optsDryRun:      false,
			expectWrite:     true,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputFile := filepath.Join(tmpDir, ".kube", "config")

			oldAllowedPaths := allowedPaths
			allowedPaths = []string{tmpDir + "/"}
			defer func() { allowedPaths = oldAllowedPaths }()

			if tt.existingContent != "" {
				err := os.MkdirAll(filepath.Dir(outputFile), 0755)
				assert.NilError(t, err)
				err = os.WriteFile(outputFile, []byte(tt.existingContent), 0644)
				assert.NilError(t, err)
			}

			masterKey := make([]byte, 32)
			encrypted, err := Encrypt([]byte("new content"), masterKey)
			assert.NilError(t, err)

			vault := &Vault{
				Secrets: []Secret{
					{
						Type:        "kubeconfig",
						Value:       encrypted,
						Destination: outputFile,
						Force:       tt.secretForce,
					},
				},
			}

			opts := WriteOptions{Force: tt.optsForce, DryRun: tt.optsDryRun}
			err = vault.DecryptAll(masterKey, opts)

			if tt.expectError {
				assert.ErrorContains(t, err, "already exists")
			} else {
				assert.NilError(t, err)
			}

			content, readErr := os.ReadFile(outputFile)
			if tt.expectWrite {
				assert.NilError(t, readErr)
				assert.Equal(t, "new content", string(content))
			} else if !tt.expectError {
				assert.Assert(t, os.IsNotExist(readErr))
			}
		})
	}
}

func TestVaultDecryptAllEmptyValue(t *testing.T) {
	vault := &Vault{
		Secrets: []Secret{
			{Type: "kubeconfig", Value: "", Destination: "/home/dev/.kube/config"},
		},
	}

	opts := WriteOptions{Force: false, DryRun: false}
	err := vault.DecryptAll(make([]byte, 32), opts)
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

	expected := []string{
		"type: kubeconfig",
		"value: encrypted_value",
		"destination: /home/dev/.kube/config",
		"force: true",
		"type: env",
		"destination: MY_VAR",
	}

	for _, exp := range expected {
		assert.Assert(t, strings.Contains(string(yamlData), exp))
	}
}

func TestVaultToYAMLEmpty(t *testing.T) {
	vault := &Vault{Secrets: []Secret{}}
	yamlData, err := vault.ToYAML()
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(yamlData), "secrets: []"))
}

func TestEndToEndVaultWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	kubeconfigSrc := filepath.Join(tmpDir, "source", ".kube", "config")
	kubeconfigDest := filepath.Join(tmpDir, "dest", ".kube", "config")

	err := os.MkdirAll(filepath.Dir(kubeconfigSrc), 0755)
	assert.NilError(t, err)
	err = os.WriteFile(kubeconfigSrc, []byte("apiVersion: v1\nkind: Config"), 0600)
	assert.NilError(t, err)

	os.Setenv("MY_SECRET_TOKEN", "super_secret_token_value")
	defer os.Unsetenv("MY_SECRET_TOKEN")

	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	vault := &Vault{
		Secrets: []Secret{
			{Type: "kubeconfig", Destination: kubeconfigSrc},
			{Type: "env", Destination: "MY_SECRET_TOKEN"},
		},
	}

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	err = vault.EncryptAll(masterKey)
	assert.NilError(t, err)

	for _, secret := range vault.Secrets {
		assert.Assert(t, strings.HasPrefix(secret.Value, "argon2id$"))
	}

	vaultYAML, err := vault.ToYAML()
	assert.NilError(t, err)

	vaultFile := filepath.Join(tmpDir, "encrypted_vault.yaml")
	err = os.WriteFile(vaultFile, vaultYAML, 0644)
	assert.NilError(t, err)

	loadedVault, err := LoadVaultFromFile(vaultFile)
	assert.NilError(t, err)
	loadedVault.Secrets[0].Destination = kubeconfigDest

	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	opts := WriteOptions{Force: false, DryRun: false}
	err = loadedVault.DecryptAll(masterKey, opts)
	assert.NilError(t, err)

	decryptedKubeconfig, err := os.ReadFile(kubeconfigDest)
	assert.NilError(t, err)
	assert.Equal(t, "apiVersion: v1\nkind: Config", string(decryptedKubeconfig))

	envFilePath := filepath.Join(tmpDir, ".zshenv")
	envFileContent, err := os.ReadFile(envFilePath)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(envFileContent), "export MY_SECRET_TOKEN=super_secret_token_value"))

	info, err := os.Stat(kubeconfigDest)
	assert.NilError(t, err)
	assert.Equal(t, FileModeKubeconfig, info.Mode().Perm())
}

func TestEndToEndMultipleSecretTypes(t *testing.T) {
	tmpDir := t.TempDir()

	files := map[string]string{
		filepath.Join(tmpDir, "source", ".kube", "config"):  "kube config",
		filepath.Join(tmpDir, "source", ".ssh", "id_rsa"):   "ssh private key",
		filepath.Join(tmpDir, "source", "password.txt"):     "my_password",
	}

	for path, content := range files {
		err := os.MkdirAll(filepath.Dir(path), 0755)
		assert.NilError(t, err)
		err = os.WriteFile(path, []byte(content), 0600)
		assert.NilError(t, err)
	}

	os.Setenv("API_TOKEN", "token123")
	os.Setenv("DATABASE_URL", "postgres://localhost/db")
	defer func() {
		os.Unsetenv("API_TOKEN")
		os.Unsetenv("DATABASE_URL")
	}()

	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	vault := &Vault{
		Secrets: []Secret{
			{Type: "kubeconfig", Destination: filepath.Join(tmpDir, "source", ".kube", "config")},
			{Type: "ssh", Destination: filepath.Join(tmpDir, "source", ".ssh", "id_rsa")},
			{Type: "password", Destination: filepath.Join(tmpDir, "source", "password.txt")},
			{Type: "env", Destination: "API_TOKEN"},
			{Type: "env", Destination: "DATABASE_URL"},
		},
	}

	masterKey := make([]byte, 32)
	err := vault.EncryptAll(masterKey)
	assert.NilError(t, err)

	expectedValues := []string{"kube config", "ssh private key", "my_password", "token123", "postgres://localhost/db"}
	for i, expected := range expectedValues {
		assert.Assert(t, strings.HasPrefix(vault.Secrets[i].Value, "argon2id$"))
		decrypted, err := Decrypt(vault.Secrets[i].Value, masterKey)
		assert.NilError(t, err)
		assert.Equal(t, expected, string(decrypted))
	}
}
