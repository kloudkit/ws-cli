package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

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

	plaintextVault := &Vault{
		Secrets: []Secret{
			{
				Type:        "kubeconfig",
				Destination: kubeconfigSrc,
			},
			{
				Type:        "env",
				Destination: "MY_SECRET_TOKEN",
			},
		},
	}

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	err = plaintextVault.EncryptAll(masterKey)
	assert.NilError(t, err)

	assert.Assert(t, strings.HasPrefix(plaintextVault.Secrets[0].Value, "argon2id$"))
	assert.Assert(t, strings.HasPrefix(plaintextVault.Secrets[1].Value, "argon2id$"))

	vaultYAML, err := plaintextVault.ToYAML()
	assert.NilError(t, err)

	vaultFile := filepath.Join(tmpDir, "encrypted_vault.yaml")
	err = os.WriteFile(vaultFile, vaultYAML, 0644)
	assert.NilError(t, err)

	loadedVault, err := LoadVaultFromFile(vaultFile)
	assert.NilError(t, err)

	loadedVault.Secrets[0].Destination = kubeconfigDest

	envFilePath := filepath.Join(tmpDir, ".zshenv")
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	opts := WriteOptions{Force: false, DryRun: false}
	err = loadedVault.DecryptAll(masterKey, opts)
	assert.NilError(t, err)

	decryptedKubeconfig, err := os.ReadFile(kubeconfigDest)
	assert.NilError(t, err)
	assert.Equal(t, "apiVersion: v1\nkind: Config", string(decryptedKubeconfig))

	envFileContent, err := os.ReadFile(envFilePath)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(envFileContent), "export MY_SECRET_TOKEN=super_secret_token_value"))

	info, err := os.Stat(kubeconfigDest)
	assert.NilError(t, err)
	assert.Equal(t, FileModeKubeconfig, info.Mode().Perm())
}

func TestEndToEndMultipleSecretTypesWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	kubeconfigFile := filepath.Join(tmpDir, "source", ".kube", "config")
	sshKeyFile := filepath.Join(tmpDir, "source", ".ssh", "id_rsa")
	passwordFile := filepath.Join(tmpDir, "source", "password.txt")

	err := os.MkdirAll(filepath.Dir(kubeconfigFile), 0755)
	assert.NilError(t, err)
	err = os.MkdirAll(filepath.Dir(sshKeyFile), 0755)
	assert.NilError(t, err)

	err = os.WriteFile(kubeconfigFile, []byte("kube config"), 0600)
	assert.NilError(t, err)
	err = os.WriteFile(sshKeyFile, []byte("ssh private key"), 0600)
	assert.NilError(t, err)
	err = os.WriteFile(passwordFile, []byte("my_password"), 0600)
	assert.NilError(t, err)

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
			{Type: "kubeconfig", Destination: kubeconfigFile},
			{Type: "ssh", Destination: sshKeyFile},
			{Type: "password", Destination: passwordFile},
			{Type: "env", Destination: "API_TOKEN"},
			{Type: "env", Destination: "DATABASE_URL"},
		},
	}

	masterKey := make([]byte, 32)
	err = vault.EncryptAll(masterKey)
	assert.NilError(t, err)

	for _, secret := range vault.Secrets {
		assert.Assert(t, strings.HasPrefix(secret.Value, "argon2id$"))
	}

	decrypted0, err := Decrypt(vault.Secrets[0].Value, masterKey)
	assert.NilError(t, err)
	assert.Equal(t, "kube config", string(decrypted0))

	decrypted1, err := Decrypt(vault.Secrets[1].Value, masterKey)
	assert.NilError(t, err)
	assert.Equal(t, "ssh private key", string(decrypted1))

	decrypted2, err := Decrypt(vault.Secrets[2].Value, masterKey)
	assert.NilError(t, err)
	assert.Equal(t, "my_password", string(decrypted2))

	decrypted3, err := Decrypt(vault.Secrets[3].Value, masterKey)
	assert.NilError(t, err)
	assert.Equal(t, "token123", string(decrypted3))

	decrypted4, err := Decrypt(vault.Secrets[4].Value, masterKey)
	assert.NilError(t, err)
	assert.Equal(t, "postgres://localhost/db", string(decrypted4))
}

func TestEndToEndForceAndDryRunWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	secretFile := filepath.Join(tmpDir, ".kube", "config")
	err := os.MkdirAll(filepath.Dir(secretFile), 0755)
	assert.NilError(t, err)
	err = os.WriteFile(secretFile, []byte("original content"), 0600)
	assert.NilError(t, err)

	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	masterKey := make([]byte, 32)
	encrypted, err := Encrypt([]byte("new content"), masterKey)
	assert.NilError(t, err)

	vault := &Vault{
		Secrets: []Secret{
			{
				Type:        "kubeconfig",
				Value:       encrypted,
				Destination: secretFile,
				Force:       false,
			},
		},
	}

	opts := WriteOptions{Force: false, DryRun: true}
	err = vault.DecryptAll(masterKey, opts)
	assert.NilError(t, err)

	content, err := os.ReadFile(secretFile)
	assert.NilError(t, err)
	assert.Equal(t, "original content", string(content))

	opts = WriteOptions{Force: false, DryRun: false}
	err = vault.DecryptAll(masterKey, opts)
	assert.ErrorContains(t, err, "already exists")

	content, err = os.ReadFile(secretFile)
	assert.NilError(t, err)
	assert.Equal(t, "original content", string(content))

	opts = WriteOptions{Force: true, DryRun: false}
	err = vault.DecryptAll(masterKey, opts)
	assert.NilError(t, err)

	content, err = os.ReadFile(secretFile)
	assert.NilError(t, err)
	assert.Equal(t, "new content", string(content))
}

func TestEndToEndVaultRoundTripWithYAML(t *testing.T) {
	tmpDir := t.TempDir()

	sourceFile := filepath.Join(tmpDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte("secret data"), 0600)
	assert.NilError(t, err)

	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	vault1 := &Vault{
		Secrets: []Secret{
			{
				Type:        "config",
				Destination: sourceFile,
				Force:       true,
			},
		},
	}

	masterKey := make([]byte, 32)
	err = vault1.EncryptAll(masterKey)
	assert.NilError(t, err)

	yamlData, err := vault1.ToYAML()
	assert.NilError(t, err)

	vaultFile := filepath.Join(tmpDir, "vault.yaml")
	err = os.WriteFile(vaultFile, yamlData, 0644)
	assert.NilError(t, err)

	vault2, err := LoadVaultFromFile(vaultFile)
	assert.NilError(t, err)

	assert.Equal(t, 1, len(vault2.Secrets))
	assert.Equal(t, "config", vault2.Secrets[0].Type)
	assert.Equal(t, sourceFile, vault2.Secrets[0].Destination)
	assert.Equal(t, true, vault2.Secrets[0].Force)
	assert.Assert(t, strings.HasPrefix(vault2.Secrets[0].Value, "argon2id$"))

	destFile := filepath.Join(tmpDir, "dest.txt")
	vault2.Secrets[0].Destination = destFile

	opts := WriteOptions{Force: false, DryRun: false}
	err = vault2.DecryptAll(masterKey, opts)
	assert.NilError(t, err)

	decrypted, err := os.ReadFile(destFile)
	assert.NilError(t, err)
	assert.Equal(t, "secret data", string(decrypted))

	info, err := os.Stat(destFile)
	assert.NilError(t, err)
	assert.Equal(t, FileModeConfig, info.Mode().Perm())
}

func TestEndToEndMixedFileAndEnvDestinations(t *testing.T) {
	tmpDir := t.TempDir()

	configFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configFile, []byte("app: myapp"), 0644)
	assert.NilError(t, err)

	os.Setenv("DB_PASSWORD", "db_secret")
	os.Setenv("API_KEY", "api_secret")
	defer func() {
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("API_KEY")
	}()

	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	vault := &Vault{
		Secrets: []Secret{
			{Type: "config", Destination: configFile},
			{Type: "env", Destination: "DB_PASSWORD"},
			{Type: "env", Destination: "API_KEY"},
		},
	}

	masterKey := make([]byte, 32)
	err = vault.EncryptAll(masterKey)
	assert.NilError(t, err)

	assert.Equal(t, 3, len(vault.Secrets))
	for _, secret := range vault.Secrets {
		assert.Assert(t, strings.HasPrefix(secret.Value, "argon2id$"))
	}

	vault.Secrets[0].Destination = filepath.Join(tmpDir, "dest_config.yaml")

	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	opts := WriteOptions{Force: false, DryRun: false}
	err = vault.DecryptAll(masterKey, opts)
	assert.NilError(t, err)

	fileContent, err := os.ReadFile(filepath.Join(tmpDir, "dest_config.yaml"))
	assert.NilError(t, err)
	assert.Equal(t, "app: myapp", string(fileContent))

	envContent, err := os.ReadFile(filepath.Join(tmpDir, ".zshenv"))
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(envContent), "export DB_PASSWORD=db_secret"))
	assert.Assert(t, strings.Contains(string(envContent), "export API_KEY=api_secret"))
}

func TestEndToEndInvalidDestinationPreventsDecrypt(t *testing.T) {
	masterKey := make([]byte, 32)
	encrypted, err := Encrypt([]byte("secret"), masterKey)
	assert.NilError(t, err)

	vault := &Vault{
		Secrets: []Secret{
			{
				Type:        "config",
				Value:       encrypted,
				Destination: "/invalid/path/file.txt",
			},
		},
	}

	opts := WriteOptions{Force: false, DryRun: false}
	err = vault.DecryptAll(masterKey, opts)
	assert.ErrorContains(t, err, "not in allowed directories")
}

func TestEndToEndEmptyVault(t *testing.T) {
	tmpDir := t.TempDir()
	vaultFile := filepath.Join(tmpDir, "empty_vault.yaml")

	vault := &Vault{
		Secrets: []Secret{},
	}

	yamlData, err := vault.ToYAML()
	assert.NilError(t, err)

	err = os.WriteFile(vaultFile, yamlData, 0644)
	assert.NilError(t, err)

	loadedVault, err := LoadVaultFromFile(vaultFile)
	assert.NilError(t, err)
	assert.Equal(t, 0, len(loadedVault.Secrets))

	masterKey := make([]byte, 32)
	opts := WriteOptions{Force: false, DryRun: false}
	err = loadedVault.DecryptAll(masterKey, opts)
	assert.NilError(t, err)
}
