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

	oldEnvFile := EnvFile
	defer func() {
		os.Unsetenv("HOME")
		_ = os.Remove(envFile)
	}()

	os.Setenv("HOME", tmpDir)

	err := WriteSecret(secret, content, opts)
	assert.NilError(t, err)

	written, err := os.ReadFile(envFile)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(written), "export MY_TEST_VAR=secret_value"))

	_ = oldEnvFile
}

func TestWriteFileWithoutForceFailsIfExists(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")

	err := os.WriteFile(testFile, []byte("existing"), 0644)
	assert.NilError(t, err)

	opts := WriteOptions{Force: false, DryRun: false}
	err = writeFile(testFile, []byte("new content"), 0644, opts)
	assert.ErrorContains(t, err, "already exists")
}

func TestWriteFileWithForceOverwrites(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")

	err := os.WriteFile(testFile, []byte("existing"), 0644)
	assert.NilError(t, err)

	opts := WriteOptions{Force: true, DryRun: false}
	err = writeFile(testFile, []byte("new content"), 0644, opts)
	assert.NilError(t, err)

	written, err := os.ReadFile(testFile)
	assert.NilError(t, err)
	assert.Equal(t, "new content", string(written))
}

func TestWriteFileDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "dryrun.txt")

	opts := WriteOptions{Force: false, DryRun: true}
	err := writeFile(testFile, []byte("content"), 0600, opts)
	assert.NilError(t, err)

	_, err = os.Stat(testFile)
	assert.Assert(t, os.IsNotExist(err))
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

func TestWriteEnvVarDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	opts := WriteOptions{Force: false, DryRun: true}
	err := writeEnvVar("TEST_VAR", "value", opts)
	assert.NilError(t, err)

	envFile := filepath.Join(tmpDir, ".zshenv")
	_, err = os.Stat(envFile)
	assert.Assert(t, os.IsNotExist(err))
}

func TestWriteEnvVarDuplicateDetection(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".zshenv")

	content := "export EXISTING_VAR=value1\nexport OTHER_VAR=value2\n"
	err := os.WriteFile(envFile, []byte(content), 0644)
	assert.NilError(t, err)

	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	opts := WriteOptions{Force: false, DryRun: false}
	err = writeEnvVar("EXISTING_VAR", "new_value", opts)
	assert.ErrorContains(t, err, "already exists")
}

func TestWriteEnvVarNewVariable(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".zshenv")

	content := "export EXISTING_VAR=value1\n"
	err := os.WriteFile(envFile, []byte(content), 0644)
	assert.NilError(t, err)

	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	opts := WriteOptions{Force: false, DryRun: false}
	err = writeEnvVar("NEW_VAR", "new_value", opts)
	assert.NilError(t, err)

	written, err := os.ReadFile(envFile)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(written), "export EXISTING_VAR=value1"))
	assert.Assert(t, strings.Contains(string(written), "export NEW_VAR=new_value"))
}

func TestEnvVarExistsFileDoesNotExist(t *testing.T) {
	exists, err := envVarExists("/nonexistent/file", "VAR")
	assert.NilError(t, err)
	assert.Equal(t, false, exists)
}

func TestEnvVarExistsTrue(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".zshenv")

	content := "export MY_VAR=value\nexport OTHER=other\n"
	err := os.WriteFile(envFile, []byte(content), 0644)
	assert.NilError(t, err)

	exists, err := envVarExists(envFile, "MY_VAR")
	assert.NilError(t, err)
	assert.Equal(t, true, exists)
}

func TestEnvVarExistsFalse(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".zshenv")

	content := "export MY_VAR=value\n"
	err := os.WriteFile(envFile, []byte(content), 0644)
	assert.NilError(t, err)

	exists, err := envVarExists(envFile, "NONEXISTENT")
	assert.NilError(t, err)
	assert.Equal(t, false, exists)
}

func TestEnvVarExistsWithWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".zshenv")

	content := "  export MY_VAR=value  \nexport OTHER=other\n"
	err := os.WriteFile(envFile, []byte(content), 0644)
	assert.NilError(t, err)

	exists, err := envVarExists(envFile, "MY_VAR")
	assert.NilError(t, err)
	assert.Equal(t, true, exists)
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
	tmpDir := t.TempDir()

	testCases := []struct {
		secretType string
		expectedMode os.FileMode
	}{
		{"kubeconfig", FileModeKubeconfig},
		{"ssh", FileModeSSH},
		{"password", FileModePassword},
		{"config", FileModeConfig},
		{"unknown", FileModeDefault},
	}

	oldAllowedPaths := allowedPaths
	allowedPaths = []string{tmpDir + "/"}
	defer func() { allowedPaths = oldAllowedPaths }()

	for _, tc := range testCases {
		t.Run(tc.secretType, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tc.secretType+"_test.txt")

			secret := &Secret{
				Type:        tc.secretType,
				Destination: testFile,
			}

			content := []byte("test content")
			opts := WriteOptions{Force: false, DryRun: false}

			err := WriteSecret(secret, content, opts)
			assert.NilError(t, err)

			info, err := os.Stat(testFile)
			assert.NilError(t, err)
			assert.Equal(t, tc.expectedMode, info.Mode().Perm())
		})
	}
}
