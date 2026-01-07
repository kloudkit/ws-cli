package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestWriteSecretToFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, ".kube", "config")

	secret := &Secret{
		Type:        "kubeconfig",
		Destination: testFile,
	}

	content := []byte("test kubeconfig content")
	opts := WriteOptions{Force: false, DryRun: false}

	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	err := WriteSecret(secret, content, opts)
	assert.NilError(t, err)

	written, err := os.ReadFile(testFile)
	assert.NilError(t, err)
	assert.Equal(t, string(content), string(written))

	info, err := os.Stat(testFile)
	assert.NilError(t, err)
	assert.Equal(t, FileModeKubeconfig, info.Mode().Perm())
}

func TestWriteSecretToEnv(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".zshenv")

	secret := &Secret{
		Type:        "env",
		Destination: "MY_TEST_VAR",
	}

	content := []byte("secret_value")
	opts := WriteOptions{Force: false, DryRun: false}

	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	err := WriteSecret(secret, content, opts)
	assert.NilError(t, err)

	written, err := os.ReadFile(envFile)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(written), "export MY_TEST_VAR=secret_value"))
}

func TestWriteFileForceAndDryRun(t *testing.T) {
	tests := []struct {
		name            string
		existingContent string
		force           bool
		dryRun          bool
		expectWrite     bool
		expectError     bool
	}{
		{
			name:        "new file without force",
			force:       false,
			dryRun:      false,
			expectWrite: true,
			expectError: false,
		},
		{
			name:            "existing file without force fails",
			existingContent: "existing",
			force:           false,
			dryRun:          false,
			expectWrite:     false,
			expectError:     true,
		},
		{
			name:            "existing file with force overwrites",
			existingContent: "existing",
			force:           true,
			dryRun:          false,
			expectWrite:     true,
			expectError:     false,
		},
		{
			name:        "dry run prevents write",
			force:       false,
			dryRun:      true,
			expectWrite: false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")

			if tt.existingContent != "" {
				err := os.WriteFile(testFile, []byte(tt.existingContent), 0644)
				assert.NilError(t, err)
			}

			opts := WriteOptions{Force: tt.force, DryRun: tt.dryRun}
			err := writeFile(testFile, []byte("new content"), 0644, opts)

			if tt.expectError {
				assert.ErrorContains(t, err, "already exists")
			} else {
				assert.NilError(t, err)
			}

			content, readErr := os.ReadFile(testFile)
			if tt.expectWrite {
				assert.NilError(t, readErr)
				assert.Equal(t, "new content", string(content))
			} else if tt.existingContent != "" && !tt.force {
				assert.Equal(t, tt.existingContent, string(content))
			} else {
				assert.Assert(t, os.IsNotExist(readErr))
			}
		})
	}
}

func TestWriteFileCreatesParentDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "nested", "deep", "file.txt")

	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	opts := WriteOptions{Force: false, DryRun: false}
	err := writeFile(testFile, []byte("content"), 0644, opts)
	assert.NilError(t, err)

	written, err := os.ReadFile(testFile)
	assert.NilError(t, err)
	assert.Equal(t, "content", string(written))
}

func TestWriteEnvVar(t *testing.T) {
	tests := []struct {
		name           string
		existingVars   string
		varName        string
		dryRun         bool
		expectWrite    bool
		expectError    bool
		errorContains  string
	}{
		{
			name:        "new variable",
			varName:     "NEW_VAR",
			dryRun:      false,
			expectWrite: true,
			expectError: false,
		},
		{
			name:         "new variable in existing file",
			existingVars: "export EXISTING_VAR=value1\n",
			varName:      "NEW_VAR",
			dryRun:       false,
			expectWrite:  true,
			expectError:  false,
		},
		{
			name:          "duplicate variable",
			existingVars:  "export EXISTING_VAR=value1\nexport OTHER_VAR=value2\n",
			varName:       "EXISTING_VAR",
			dryRun:        false,
			expectWrite:   false,
			expectError:   true,
			errorContains: "already exists",
		},
		{
			name:        "dry run",
			varName:     "TEST_VAR",
			dryRun:      true,
			expectWrite: false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			envFile := filepath.Join(tmpDir, ".zshenv")

			if tt.existingVars != "" {
				err := os.WriteFile(envFile, []byte(tt.existingVars), 0644)
				assert.NilError(t, err)
			}

			os.Setenv("HOME", tmpDir)
			defer os.Unsetenv("HOME")

			opts := WriteOptions{Force: false, DryRun: tt.dryRun}
			err := writeEnvVar(tt.varName, "new_value", opts)

			if tt.expectError {
				assert.ErrorContains(t, err, tt.errorContains)
			} else {
				assert.NilError(t, err)
			}

			content, readErr := os.ReadFile(envFile)
			if tt.expectWrite {
				assert.NilError(t, readErr)
				assert.Assert(t, strings.Contains(string(content), "export "+tt.varName+"=new_value"))
			} else if tt.dryRun {
				if tt.existingVars == "" {
					assert.Assert(t, os.IsNotExist(readErr))
				}
			}
		})
	}
}

func TestEnvVarExists(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		varName  string
		expected bool
	}{
		{
			name:     "file does not exist",
			varName:  "VAR",
			expected: false,
		},
		{
			name:     "variable exists",
			content:  "export MY_VAR=value\nexport OTHER=other\n",
			varName:  "MY_VAR",
			expected: true,
		},
		{
			name:     "variable does not exist",
			content:  "export MY_VAR=value\n",
			varName:  "NONEXISTENT",
			expected: false,
		},
		{
			name:     "variable with whitespace",
			content:  "  export MY_VAR=value  \nexport OTHER=other\n",
			varName:  "MY_VAR",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var envFile string
			if tt.content != "" {
				tmpDir := t.TempDir()
				envFile = filepath.Join(tmpDir, ".zshenv")
				err := os.WriteFile(envFile, []byte(tt.content), 0644)
				assert.NilError(t, err)
			} else {
				envFile = "/nonexistent/file"
			}

			exists, err := envVarExists(envFile, tt.varName)
			assert.NilError(t, err)
			assert.Equal(t, tt.expected, exists)
		})
	}
}

func TestWriteSecretInvalidDestination(t *testing.T) {
	secret := &Secret{
		Type:        "kubeconfig",
		Destination: "/invalid/path/config",
	}

	content := []byte("content")
	opts := WriteOptions{Force: false, DryRun: false}

	err := WriteSecret(secret, content, opts)
	assert.ErrorContains(t, err, "not in allowed directories")
}

func TestWriteSecretWithFileMode(t *testing.T) {
	tests := []struct {
		secretType   string
		expectedMode os.FileMode
	}{
		{"kubeconfig", FileModeKubeconfig},
		{"ssh", FileModeSSH},
		{"password", FileModePassword},
		{"config", FileModeConfig},
		{"unknown", FileModeDefault},
	}

	tmpDir := t.TempDir()
	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	for _, tt := range tests {
		t.Run(tt.secretType, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.secretType+"_test.txt")

			secret := &Secret{
				Type:        tt.secretType,
				Destination: testFile,
			}

			content := []byte("test content")
			opts := WriteOptions{Force: false, DryRun: false}

			err := WriteSecret(secret, content, opts)
			assert.NilError(t, err)

			info, err := os.Stat(testFile)
			assert.NilError(t, err)
			assert.Equal(t, tt.expectedMode, info.Mode().Perm())
		})
	}
}
