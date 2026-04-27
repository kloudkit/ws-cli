package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestLoadVault(t *testing.T) {
	t.Run("ValidVault", func(t *testing.T) {
		vaultContent := `
secrets:
  db_password:
    encrypted: "test$encrypted"
    destination: "/etc/db/password"
  ssh_key:
    type: "ssh"
    encrypted: "test$encrypted"
    destination: "/home/user/.ssh/id_rsa"
`
		vaultFile := filepath.Join(t.TempDir(), "vault.yaml")
		err := os.WriteFile(vaultFile, []byte(vaultContent), 0600)
		assert.NilError(t, err)

		vault, err := LoadVault(vaultFile)
		assert.NilError(t, err)
		assert.Equal(t, 2, len(vault.Secrets))
		assert.Equal(t, TypeGeneric, vault.Secrets["db_password"].Type)
		assert.Equal(t, TypeSSH, vault.Secrets["ssh_key"].Type)
		assert.Equal(t, "0o600", vault.Secrets["db_password"].Mode)
	})

	t.Run("EmptyVault", func(t *testing.T) {
		vaultContent := `secrets: {}`
		vaultFile := filepath.Join(t.TempDir(), "vault.yaml")
		err := os.WriteFile(vaultFile, []byte(vaultContent), 0600)
		assert.NilError(t, err)

		vault, err := LoadVault(vaultFile)
		assert.NilError(t, err)
		assert.Equal(t, 0, len(vault.Secrets))
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		vaultContent := `invalid: yaml: content:`
		vaultFile := filepath.Join(t.TempDir(), "vault.yaml")
		err := os.WriteFile(vaultFile, []byte(vaultContent), 0600)
		assert.NilError(t, err)

		_, err = LoadVault(vaultFile)
		assert.ErrorContains(t, err, "failed to unmarshal")
	})

	t.Run("FileNotFound", func(t *testing.T) {
		_, err := LoadVault("/nonexistent/vault.yaml")
		assert.ErrorContains(t, err, "failed to read vault file")
	})

	t.Run("RelativePathResolution", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		assert.NilError(t, err)

		vaultContent := `
secrets:
  ssh_key:
    type: "ssh"
    encrypted: "test$encrypted"
    destination: "github.com/id_ed25519"
  kubeconfig:
    type: "kubeconfig"
    encrypted: "test$encrypted"
    destination: "config"
`
		vaultFile := filepath.Join(t.TempDir(), "vault.yaml")
		err = os.WriteFile(vaultFile, []byte(vaultContent), 0600)
		assert.NilError(t, err)

		vault, err := LoadVault(vaultFile)
		assert.NilError(t, err)
		assert.Equal(t, 2, len(vault.Secrets))
		assert.Equal(t, filepath.Join(homeDir, ".ssh/github.com/id_ed25519"), vault.Secrets["ssh_key"].Destination)
		assert.Equal(t, filepath.Join(homeDir, ".kube/config"), vault.Secrets["kubeconfig"].Destination)
	})

	t.Run("GenericTypeRequiresAbsolute", func(t *testing.T) {
		vaultContent := `
secrets:
  generic_secret:
    type: "generic"
    encrypted: "test$encrypted"
    destination: "relative/path"
`
		vaultFile := filepath.Join(t.TempDir(), "vault.yaml")
		err := os.WriteFile(vaultFile, []byte(vaultContent), 0600)
		assert.NilError(t, err)

		_, err = LoadVault(vaultFile)
		assert.ErrorContains(t, err, "requires an absolute path")
	})
}

func TestValidateSecret(t *testing.T) {
	tests := []struct {
		name          string
		secretName    string
		secret        VaultSecret
		errorContains string
	}{
		{
			name:       "Valid",
			secretName: "test",
			secret: VaultSecret{
				Type:        TypeGeneric,
				Encrypted:   "encrypted$value",
				Destination: "/etc/test",
			},
			errorContains: "",
		},
		{
			name:       "MissingEncrypted",
			secretName: "test",
			secret: VaultSecret{
				Type:        TypeGeneric,
				Destination: "/etc/test",
			},
			errorContains: "encrypted value is required",
		},
		{
			name:       "MissingDestination",
			secretName: "test",
			secret: VaultSecret{
				Type:      TypeGeneric,
				Encrypted: "encrypted$value",
			},
			errorContains: "destination is required",
		},
		{
			name:       "InvalidType",
			secretName: "test",
			secret: VaultSecret{
				Type:        "invalid",
				Encrypted:   "encrypted$value",
				Destination: "/etc/test",
			},
			errorContains: "invalid type",
		},
		{
			name:       "RelativePathGenericType",
			secretName: "test",
			secret: VaultSecret{
				Type:        TypeGeneric,
				Encrypted:   "encrypted$value",
				Destination: "relative/path",
			},
			errorContains: "invalid destination path",
		},
		{
			name:       "EnvTypeValid",
			secretName: "test",
			secret: VaultSecret{
				Type:        TypeEnv,
				Encrypted:   "encrypted$value",
				Destination: "MY_VAR",
			},
			errorContains: "",
		},
		{
			name:       "EnvTypeValidUnderscore",
			secretName: "test",
			secret: VaultSecret{
				Type:        TypeEnv,
				Encrypted:   "encrypted$value",
				Destination: "_MY_VAR",
			},
			errorContains: "",
		},
		{
			name:       "EnvTypeValidWithNumbers",
			secretName: "test",
			secret: VaultSecret{
				Type:        TypeEnv,
				Encrypted:   "encrypted$value",
				Destination: "MY_VAR_123",
			},
			errorContains: "",
		},
		{
			name:       "EnvTypeInvalidStartsWithNumber",
			secretName: "test",
			secret: VaultSecret{
				Type:        TypeEnv,
				Encrypted:   "encrypted$value",
				Destination: "123_VAR",
			},
			errorContains: "invalid environment variable name",
		},
		{
			name:       "EnvTypeInvalidHyphen",
			secretName: "test",
			secret: VaultSecret{
				Type:        TypeEnv,
				Encrypted:   "encrypted$value",
				Destination: "MY-VAR",
			},
			errorContains: "invalid environment variable name",
		},
		{
			name:       "EnvTypeInvalidDot",
			secretName: "test",
			secret: VaultSecret{
				Type:        TypeEnv,
				Encrypted:   "encrypted$value",
				Destination: "MY.VAR",
			},
			errorContains: "invalid environment variable name",
		},
		{
			name:       "EnvTypeInvalidSpace",
			secretName: "test",
			secret: VaultSecret{
				Type:        TypeEnv,
				Encrypted:   "encrypted$value",
				Destination: "MY VAR",
			},
			errorContains: "invalid environment variable name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSecret(tt.secretName, tt.secret)
			if tt.errorContains != "" {
				assert.ErrorContains(t, err, tt.errorContains)
			} else {
				assert.NilError(t, err)
			}
		})
	}
}

func TestGetSecretKeys(t *testing.T) {
	vault := &Vault{
		Secrets: map[string]VaultSecret{
			"key1": {},
			"key2": {},
			"key3": {},
		},
	}

	t.Run("AllKeys", func(t *testing.T) {
		keys := GetSecretKeys(vault, []string{})
		assert.Equal(t, 3, len(keys))
		for i := 1; i < len(keys); i++ {
			assert.Assert(t, keys[i-1] < keys[i], "keys should be sorted alphabetically")
		}
	})

	t.Run("SpecificKeys", func(t *testing.T) {
		keys := GetSecretKeys(vault, []string{"key1", "key3"})
		assert.Equal(t, 2, len(keys))
		assert.Equal(t, "key1", keys[0])
		assert.Equal(t, "key3", keys[1])
	})
}

func TestResolveVaultPath(t *testing.T) {
	t.Run("FromFlag", func(t *testing.T) {
		path, err := ResolveVaultPath("/path/to/vault.yaml")
		assert.NilError(t, err)
		assert.Equal(t, "/path/to/vault.yaml", path)
	})

	t.Run("FromEnv", func(t *testing.T) {
		t.Setenv("WS_SECRETS_VAULT", "/env/vault.yaml")
		path, err := ResolveVaultPath("")
		assert.NilError(t, err)
		assert.Equal(t, "/env/vault.yaml", path)
	})

	t.Run("FromDefault", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("WS_SECRETS_VAULT", "")

		defaultVault := filepath.Join(home, ".ws", "vault", "secrets.yaml")
		assert.NilError(t, os.MkdirAll(filepath.Dir(defaultVault), 0o755))
		assert.NilError(t, os.WriteFile(defaultVault, []byte("secrets:\n"), 0o600))

		path, err := ResolveVaultPath("")
		assert.NilError(t, err)
		assert.Equal(t, defaultVault, path)
	})

	t.Run("NotSpecified", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("WS_SECRETS_VAULT", "")
		_, err := ResolveVaultPath("")
		assert.ErrorContains(t, err, "vault file not specified")
	})
}

func TestFormatSecretForStdout(t *testing.T) {
	t.Run("Raw", func(t *testing.T) {
		output := FormatSecretForStdout("key", "value", true)
		assert.Equal(t, "value", output)
	})

	t.Run("Formatted", func(t *testing.T) {
		output := FormatSecretForStdout("key", "value", false)
		assert.Equal(t, "[key]\nvalue\n", output)
	})
}

func TestResolveDestination(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	assert.NilError(t, err)

	tests := []struct {
		name          string
		secret        VaultSecret
		expected      string
		errorContains string
	}{
		{
			name: "EnvType",
			secret: VaultSecret{
				Type:        TypeEnv,
				Destination: "MY_VAR",
			},
			expected: "MY_VAR",
		},
		{
			name: "SSHRelativePath",
			secret: VaultSecret{
				Type:        TypeSSH,
				Destination: "github.com/id_ed25519",
			},
			expected: filepath.Join(homeDir, ".ssh", "github.com/id_ed25519"),
		},
		{
			name: "SSHAbsolutePath",
			secret: VaultSecret{
				Type:        TypeSSH,
				Destination: "/custom/path/key",
			},
			expected: "/custom/path/key",
		},
		{
			name: "SSHTildePath",
			secret: VaultSecret{
				Type:        TypeSSH,
				Destination: "~/.ssh/custom/key",
			},
			expected: filepath.Join(homeDir, ".ssh/custom/key"),
		},
		{
			name: "KubeconfigRelativePath",
			secret: VaultSecret{
				Type:        TypeKubeconfig,
				Destination: "config",
			},
			expected: filepath.Join(homeDir, ".kube/config"),
		},
		{
			name: "DockerConfigJSONRelativePath",
			secret: VaultSecret{
				Type:        TypeDockerConfigJSON,
				Destination: "config.json",
			},
			expected: filepath.Join(homeDir, ".docker/config.json"),
		},
		{
			name: "GenericAbsolutePath",
			secret: VaultSecret{
				Type:        TypeGeneric,
				Destination: "/etc/secret",
			},
			expected: "/etc/secret",
		},
		{
			name: "GenericRelativePath",
			secret: VaultSecret{
				Type:        TypeGeneric,
				Destination: "relative",
			},
			errorContains: "requires an absolute path",
		},
		{
			name: "UnknownType",
			secret: VaultSecret{
				Type:        "unknown",
				Destination: "/etc/secret",
			},
			errorContains: "unknown type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveDestination(tt.secret)
			if tt.errorContains != "" {
				assert.ErrorContains(t, err, tt.errorContains)
			} else {
				assert.NilError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestProcessEnvSecret(t *testing.T) {
	t.Run("NewVariable", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".zshenv")

		t.Setenv("HOME", tmpDir)

		err := ProcessEnvSecret("NEW_VAR", []byte("secret_value"), false)
		assert.NilError(t, err)

		content, err := os.ReadFile(envFile)
		assert.NilError(t, err)
		assert.Assert(t, len(content) > 0)
	})

	t.Run("ExistingWithoutForce", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".zshenv")

		t.Setenv("HOME", tmpDir)

		initialContent := `export EXISTING_VAR="old_value"
`
		err := os.WriteFile(envFile, []byte(initialContent), 0644)
		assert.NilError(t, err)

		err = ProcessEnvSecret("EXISTING_VAR", []byte("new_value"), false)
		assert.ErrorContains(t, err, "already exists")
	})

	t.Run("ExistingWithForce", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".zshenv")

		t.Setenv("HOME", tmpDir)

		initialContent := `export EXISTING_VAR="old_value"
`
		err := os.WriteFile(envFile, []byte(initialContent), 0644)
		assert.NilError(t, err)

		err = ProcessEnvSecret("EXISTING_VAR", []byte("new_value"), true)
		assert.NilError(t, err)

		content, err := os.ReadFile(envFile)
		assert.NilError(t, err)
		assert.Assert(t, len(content) > 0)
	})

	t.Run("MultipleCalls", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".zshenv")

		t.Setenv("HOME", tmpDir)

		err := ProcessEnvSecret("VAR1", []byte("value1"), false)
		assert.NilError(t, err)

		err = ProcessEnvSecret("VAR2", []byte("value2"), false)
		assert.NilError(t, err)

		content, err := os.ReadFile(envFile)
		assert.NilError(t, err)

		contentStr := string(content)
		assert.Assert(t, strings.Contains(contentStr, `export VAR1="value1"`))
		assert.Assert(t, strings.Contains(contentStr, `export VAR2="value2"`))
	})

	t.Run("DuplicateCallWithoutForce", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".zshenv")

		t.Setenv("HOME", tmpDir)

		err := ProcessEnvSecret("DUPLICATE_VAR", []byte("value1"), false)
		assert.NilError(t, err)

		err = ProcessEnvSecret("DUPLICATE_VAR", []byte("value2"), false)
		assert.ErrorContains(t, err, "already exists")

		content, err := os.ReadFile(envFile)
		assert.NilError(t, err)
		assert.Assert(t, strings.Contains(string(content), `export DUPLICATE_VAR="value1"`))
		assert.Assert(t, !strings.Contains(string(content), "value2"))
	})

	t.Run("DuplicateCallWithForce", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".zshenv")

		t.Setenv("HOME", tmpDir)

		err := ProcessEnvSecret("DUPLICATE_VAR", []byte("value1"), false)
		assert.NilError(t, err)

		err = ProcessEnvSecret("DUPLICATE_VAR", []byte("value2"), true)
		assert.NilError(t, err)

		content, err := os.ReadFile(envFile)
		assert.NilError(t, err)

		contentStr := string(content)
		assert.Assert(t, strings.Contains(contentStr, `export DUPLICATE_VAR="value2"`))
		assert.Assert(t, !strings.Contains(contentStr, "value1"))

		lines := strings.Split(strings.TrimSpace(contentStr), "\n")
		assert.Equal(t, 1, len(lines))
	})
}

func TestDeterministicOrdering(t *testing.T) {
	t.Run("GetSecretKeysReturnsSorted", func(t *testing.T) {
		vault := &Vault{
			Secrets: map[string]VaultSecret{
				"zebra":   {},
				"alpha":   {},
				"charlie": {},
				"bravo":   {},
			},
		}

		keys := GetSecretKeys(vault, []string{})
		assert.Equal(t, 4, len(keys))
		assert.Equal(t, "alpha", keys[0])
		assert.Equal(t, "bravo", keys[1])
		assert.Equal(t, "charlie", keys[2])
		assert.Equal(t, "zebra", keys[3])
	})

	t.Run("MultipleRunsProduceSameOrder", func(t *testing.T) {
		vault := &Vault{
			Secrets: map[string]VaultSecret{
				"secret3": {},
				"secret1": {},
				"secret2": {},
			},
		}

		firstRun := GetSecretKeys(vault, []string{})
		secondRun := GetSecretKeys(vault, []string{})
		thirdRun := GetSecretKeys(vault, []string{})

		assert.DeepEqual(t, firstRun, secondRun)
		assert.DeepEqual(t, secondRun, thirdRun)
	})

	t.Run("RequestedKeysPreserveOrder", func(t *testing.T) {
		vault := &Vault{
			Secrets: map[string]VaultSecret{
				"zebra":   {},
				"alpha":   {},
				"charlie": {},
			},
		}

		requested := []string{"zebra", "alpha"}
		keys := GetSecretKeys(vault, requested)
		assert.DeepEqual(t, requested, keys)
	})
}

func TestPerSecretForce(t *testing.T) {
	t.Run("SecretForceOverwritesEnv", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".zshenv")
		t.Setenv("HOME", tmpDir)

		existingContent := `export TEST_VAR="old_value"
`
		err := os.WriteFile(envFile, []byte(existingContent), 0644)
		assert.NilError(t, err)

		err = ProcessEnvSecret("TEST_VAR", []byte("new_value"), true)
		assert.NilError(t, err)

		content, err := os.ReadFile(envFile)
		assert.NilError(t, err)

		contentStr := string(content)
		assert.Assert(t, strings.Contains(contentStr, `export TEST_VAR="new_value"`))
		assert.Assert(t, !strings.Contains(contentStr, "old_value"))
	})

	t.Run("SecretWithoutForceFailsOnExisting", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".zshenv")
		t.Setenv("HOME", tmpDir)

		existingContent := `export TEST_VAR="old_value"
`
		err := os.WriteFile(envFile, []byte(existingContent), 0644)
		assert.NilError(t, err)

		err = ProcessEnvSecret("TEST_VAR", []byte("new_value"), false)
		assert.ErrorContains(t, err, `environment variable "TEST_VAR" already exists, use --force to overwrite`)
	})

	t.Run("MixedForceInVault", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".zshenv")
		t.Setenv("HOME", tmpDir)

		existingContent := `export VAR1="old1"
export VAR2="old2"
`
		err := os.WriteFile(envFile, []byte(existingContent), 0644)
		assert.NilError(t, err)

		err = ProcessEnvSecret("VAR1", []byte("new1"), true)
		assert.NilError(t, err)

		err = ProcessEnvSecret("VAR2", []byte("new2"), false)
		assert.ErrorContains(t, err, `environment variable "VAR2" already exists, use --force to overwrite`)

		content, err := os.ReadFile(envFile)
		assert.NilError(t, err)

		contentStr := string(content)
		assert.Assert(t, strings.Contains(contentStr, `export VAR1="new1"`))
		assert.Assert(t, strings.Contains(contentStr, `export VAR2="old2"`))
	})
}

func TestListVault(t *testing.T) {
	t.Run("SortedOutput", func(t *testing.T) {
		vault := &Vault{
			Secrets: map[string]VaultSecret{
				"zebra": {Type: TypeSSH, Destination: "/z", Encrypted: "enc"},
				"alpha": {Type: TypeGeneric, Destination: "/a", Encrypted: "enc"},
				"mike":  {Type: TypeEnv, Destination: "MY_VAR", Encrypted: "enc"},
			},
		}

		entries := ListVault(vault)
		assert.Equal(t, 3, len(entries))
		assert.Equal(t, "alpha", entries[0].Name)
		assert.Equal(t, "mike", entries[1].Name)
		assert.Equal(t, "zebra", entries[2].Name)
	})

	t.Run("EmptyVault", func(t *testing.T) {
		vault := &Vault{Secrets: map[string]VaultSecret{}}
		entries := ListVault(vault)
		assert.Equal(t, 0, len(entries))
	})

	t.Run("PreservesTypeAndDestination", func(t *testing.T) {
		vault := &Vault{
			Secrets: map[string]VaultSecret{
				"ssh_key": {Type: TypeSSH, Destination: "/home/.ssh/id_rsa", Encrypted: "enc"},
			},
		}

		entries := ListVault(vault)
		assert.Equal(t, 1, len(entries))
		assert.Equal(t, TypeSSH, entries[0].Type)
		assert.Equal(t, "/home/.ssh/id_rsa", entries[0].Destination)
	})
}

func TestLoadRawVault(t *testing.T) {
	t.Run("DoesNotResolveDestinations", func(t *testing.T) {
		vaultContent := `
secrets:
  ssh_key:
    type: "ssh"
    encrypted: "test$encrypted"
    destination: "github.com/id_ed25519"
`
		vaultFile := filepath.Join(t.TempDir(), "vault.yaml")
		err := os.WriteFile(vaultFile, []byte(vaultContent), 0600)
		assert.NilError(t, err)

		vault, err := LoadRawVault(vaultFile)
		assert.NilError(t, err)
		assert.Equal(t, "github.com/id_ed25519", vault.Secrets["ssh_key"].Destination)
	})

	t.Run("DoesNotFillDefaults", func(t *testing.T) {
		vaultContent := `
secrets:
  db_password:
    encrypted: "test$encrypted"
    destination: "/etc/db/password"
`
		vaultFile := filepath.Join(t.TempDir(), "vault.yaml")
		err := os.WriteFile(vaultFile, []byte(vaultContent), 0600)
		assert.NilError(t, err)

		vault, err := LoadRawVault(vaultFile)
		assert.NilError(t, err)
		assert.Equal(t, "", vault.Secrets["db_password"].Type)
		assert.Equal(t, "", vault.Secrets["db_password"].Mode)
	})

	t.Run("EmptyVault", func(t *testing.T) {
		vaultContent := `secrets: {}`
		vaultFile := filepath.Join(t.TempDir(), "vault.yaml")
		err := os.WriteFile(vaultFile, []byte(vaultContent), 0600)
		assert.NilError(t, err)

		vault, err := LoadRawVault(vaultFile)
		assert.NilError(t, err)
		assert.Equal(t, 0, len(vault.Secrets))
	})

	t.Run("FileNotFound", func(t *testing.T) {
		_, err := LoadRawVault("/nonexistent/vault.yaml")
		assert.ErrorContains(t, err, "failed to read vault file")
	})
}

func TestRotateVault(t *testing.T) {
	t.Run("RotateDecryptsWithNewKey", func(t *testing.T) {
		oldKey := []byte("old-master-key-for-testing")
		newKey := []byte("new-master-key-for-testing")

		encrypted, err := Encrypt([]byte("my-secret-value"), oldKey)
		assert.NilError(t, err)

		vault := &Vault{
			Secrets: map[string]VaultSecret{
				"test_secret": {
					Type:        TypeGeneric,
					Encrypted:   encrypted,
					Destination: "/etc/test",
				},
			},
		}

		fileRefs, err := RotateVault(vault, oldKey, newKey)
		assert.NilError(t, err)
		assert.Equal(t, 0, len(fileRefs))

		decrypted, err := Decrypt(vault.Secrets["test_secret"].Encrypted, newKey)
		assert.NilError(t, err)
		assert.Equal(t, "my-secret-value", string(decrypted))
	})

	t.Run("OldKeyNoLongerWorks", func(t *testing.T) {
		oldKey := []byte("old-master-key-for-testing")
		newKey := []byte("new-master-key-for-testing")

		encrypted, err := Encrypt([]byte("my-secret-value"), oldKey)
		assert.NilError(t, err)

		vault := &Vault{
			Secrets: map[string]VaultSecret{
				"test_secret": {
					Type:        TypeGeneric,
					Encrypted:   encrypted,
					Destination: "/etc/test",
				},
			},
		}

		_, err = RotateVault(vault, oldKey, newKey)
		assert.NilError(t, err)

		_, err = Decrypt(vault.Secrets["test_secret"].Encrypted, oldKey)
		assert.ErrorContains(t, err, "cipher: message authentication failed")
	})

	t.Run("FileRefsReported", func(t *testing.T) {
		oldKey := []byte("old-master-key-for-testing")
		newKey := []byte("new-master-key-for-testing")

		encrypted, err := Encrypt([]byte("file-secret"), oldKey)
		assert.NilError(t, err)

		encFile := filepath.Join(t.TempDir(), "secret.enc")
		err = os.WriteFile(encFile, []byte(encrypted), 0600)
		assert.NilError(t, err)

		vault := &Vault{
			Secrets: map[string]VaultSecret{
				"file_secret": {
					Type:        TypeGeneric,
					Encrypted:   "file:" + encFile,
					Destination: "/etc/test",
				},
			},
		}

		fileRefs, err := RotateVault(vault, oldKey, newKey)
		assert.NilError(t, err)
		assert.Equal(t, 1, len(fileRefs))
		assert.Equal(t, "file_secret", fileRefs[0])

		assert.Assert(t, !strings.HasPrefix(vault.Secrets["file_secret"].Encrypted, "file:"))
	})

	t.Run("MultipleSecrets", func(t *testing.T) {
		oldKey := []byte("old-master-key-for-testing")
		newKey := []byte("new-master-key-for-testing")

		enc1, err := Encrypt([]byte("secret-1"), oldKey)
		assert.NilError(t, err)
		enc2, err := Encrypt([]byte("secret-2"), oldKey)
		assert.NilError(t, err)

		vault := &Vault{
			Secrets: map[string]VaultSecret{
				"first":  {Encrypted: enc1, Destination: "/etc/first"},
				"second": {Encrypted: enc2, Destination: "/etc/second"},
			},
		}

		_, err = RotateVault(vault, oldKey, newKey)
		assert.NilError(t, err)

		dec1, err := Decrypt(vault.Secrets["first"].Encrypted, newKey)
		assert.NilError(t, err)
		assert.Equal(t, "secret-1", string(dec1))

		dec2, err := Decrypt(vault.Secrets["second"].Encrypted, newKey)
		assert.NilError(t, err)
		assert.Equal(t, "secret-2", string(dec2))
	})
}

func TestSaveVault(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		vault := &Vault{
			Secrets: map[string]VaultSecret{
				"db_password": {
					Type:        TypeGeneric,
					Encrypted:   "salt$cipher",
					Destination: "/etc/db/password",
					Mode:        "0o600",
				},
			},
		}

		vaultFile := filepath.Join(t.TempDir(), "vault.yaml")
		err := SaveVault(vaultFile, vault)
		assert.NilError(t, err)

		loaded, err := LoadRawVault(vaultFile)
		assert.NilError(t, err)
		assert.Equal(t, 1, len(loaded.Secrets))
		assert.Equal(t, "salt$cipher", loaded.Secrets["db_password"].Encrypted)
		assert.Equal(t, "/etc/db/password", loaded.Secrets["db_password"].Destination)
		assert.Equal(t, TypeGeneric, loaded.Secrets["db_password"].Type)
		assert.Equal(t, "0o600", loaded.Secrets["db_password"].Mode)
	})

	t.Run("FilePermissions", func(t *testing.T) {
		vault := &Vault{Secrets: map[string]VaultSecret{}}
		vaultFile := filepath.Join(t.TempDir(), "vault.yaml")

		err := SaveVault(vaultFile, vault)
		assert.NilError(t, err)

		info, err := os.Stat(vaultFile)
		assert.NilError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})

	t.Run("EmptyVault", func(t *testing.T) {
		vault := &Vault{Secrets: map[string]VaultSecret{}}
		vaultFile := filepath.Join(t.TempDir(), "vault.yaml")

		err := SaveVault(vaultFile, vault)
		assert.NilError(t, err)

		loaded, err := LoadRawVault(vaultFile)
		assert.NilError(t, err)
		assert.Equal(t, 0, len(loaded.Secrets))
	})
}
