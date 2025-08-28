package path_test

import (
	"os"
	"testing"

	"github.com/kloudkit/ws-cli/internals/path"
	"gotest.tools/v3/assert"
)

func TestAppendSegments(t *testing.T) {
	t.Run("AdditionalSegments", func(t *testing.T) {
		assert.Equal(t, "/path", path.AppendSegments("/", "path"))
	})

	t.Run("NormalizeAdditionalSegments", func(t *testing.T) {
		assert.Equal(t, "/home/sub/path", path.AppendSegments("/home/", "/sub", "path/"))
	})
}

func TestGetHomeDirectory(t *testing.T) {
	t.Run("WithEnv", func(t *testing.T) {
		t.Setenv("HOME", "/app")

		assert.Equal(t, "/app", path.GetHomeDirectory())
	})

	t.Run("WithoutEnv", func(t *testing.T) {
		os.Unsetenv("HOME")

		assert.Equal(t, "/home/kloud", path.GetHomeDirectory())
	})
}

func TestResolveConfigPath(t *testing.T) {
	t.Run("AbsolutePath", func(t *testing.T) {
		result := path.ResolveConfigPath("/etc/config")
		assert.Equal(t, "/etc/config", result)
	})

	t.Run("RelativePath", func(t *testing.T) {
		t.Setenv("HOME", "/home/user")
		result := path.ResolveConfigPath(".config/app/config")
		assert.Equal(t, "/home/user/.config/app/config", result)
	})
}

func TestFileExists(t *testing.T) {
	t.Run("ExistingFile", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := path.AppendSegments(tempDir, "test.txt")

		err := os.WriteFile(testFile, []byte("test"), 0644)
		assert.NilError(t, err)

		assert.Assert(t, path.FileExists(testFile))
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		result := path.FileExists("/non/existent/file")
		assert.Assert(t, !result)
	})
}

func TestGetCurrentWorkingDirectory(t *testing.T) {
	t.Run("WithoutSegments", func(t *testing.T) {
		result, err := path.GetCurrentWorkingDirectory()
		assert.NilError(t, err)

		cwd, err := os.Getwd()
		assert.NilError(t, err)
		assert.Equal(t, result, cwd)
	})

	t.Run("WithSegments", func(t *testing.T) {
		result, err := path.GetCurrentWorkingDirectory("sub", "path")
		assert.NilError(t, err)

		cwd, err := os.Getwd()
		assert.NilError(t, err)
		expected := path.AppendSegments(cwd, "sub", "path")
		assert.Equal(t, result, expected)
	})
}

func TestCopyFile(t *testing.T) {
	t.Run("SuccessfulCopy", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceFile := path.AppendSegments(tempDir, "source.txt")
		destFile := path.AppendSegments(tempDir, "dest.txt")
		content := "test content"

		err := os.WriteFile(sourceFile, []byte(content), 0644)
		assert.NilError(t, err)

		err = path.CopyFile(sourceFile, destFile)
		assert.NilError(t, err)

		destContent, err := os.ReadFile(destFile)
		assert.NilError(t, err)
		assert.Equal(t, string(destContent), content)
	})

	t.Run("NonExistentSource", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceFile := path.AppendSegments(tempDir, "nonexistent.txt")
		destFile := path.AppendSegments(tempDir, "dest.txt")

		err := path.CopyFile(sourceFile, destFile)
		assert.ErrorContains(t, err, "failed to stat source file")
	})

	t.Run("DirectoryAsSource", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceDir := path.AppendSegments(tempDir, "sourcedir")
		destFile := path.AppendSegments(tempDir, "dest.txt")

		err := os.Mkdir(sourceDir, 0755)
		assert.NilError(t, err)

		err = path.CopyFile(sourceDir, destFile)
		assert.ErrorContains(t, err, "is not a regular file")
	})
}
