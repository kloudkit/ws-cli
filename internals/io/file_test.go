package io

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestWriteSecureFile(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("writes new file successfully", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "new-file.txt")
		content := []byte("test content")

		err := WriteSecureFile(filePath, content, "0o600", false)
		assert.NilError(t, err)

		data, err := os.ReadFile(filePath)
		assert.NilError(t, err)
		assert.DeepEqual(t, content, data)

		info, err := os.Stat(filePath)
		assert.NilError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	})

	t.Run("fails when file exists without force", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "existing-file.txt")
		os.WriteFile(filePath, []byte("original"), 0644)

		err := WriteSecureFile(filePath, []byte("new content"), "0o600", false)
		assert.Assert(t, err != nil)
		assert.ErrorContains(t, err, "exists, use --force to overwrite")

		data, _ := os.ReadFile(filePath)
		assert.DeepEqual(t, []byte("original"), data)
	})

	t.Run("overwrites file when force is true", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "force-file.txt")
		os.WriteFile(filePath, []byte("original"), 0644)

		err := WriteSecureFile(filePath, []byte("new content"), "0o600", true)
		assert.NilError(t, err)

		data, err := os.ReadFile(filePath)
		assert.NilError(t, err)
		assert.DeepEqual(t, []byte("new content"), data)
	})

	t.Run("fails with invalid mode", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "invalid-mode.txt")

		err := WriteSecureFile(filePath, []byte("content"), "999", false)
		assert.Assert(t, err != nil)
	})

	t.Run("handles different file modes", func(t *testing.T) {
		modes := []string{"0o400", "0o600", "0o644", "0o755"}

		for _, mode := range modes {
			filePath := filepath.Join(tempDir, "mode-"+mode+".txt")
			err := WriteSecureFile(filePath, []byte("test"), mode, false)
			assert.NilError(t, err)
		}
	})

	t.Run("handles empty content", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "empty.txt")

		err := WriteSecureFile(filePath, []byte(""), "0o600", false)
		assert.NilError(t, err)

		data, err := os.ReadFile(filePath)
		assert.NilError(t, err)
		assert.DeepEqual(t, []byte(""), data)
	})

	t.Run("creates nested directories", func(t *testing.T) {
		nestedPath := filepath.Join(tempDir, "nested", "deep", "dirs", "file.txt")
		content := []byte("nested content")

		err := WriteSecureFile(nestedPath, content, "0o600", false)
		assert.NilError(t, err)

		data, err := os.ReadFile(nestedPath)
		assert.NilError(t, err)
		assert.DeepEqual(t, content, data)
	})

	t.Run("directory mode derived from file mode 0o600", func(t *testing.T) {
		nestedPath := filepath.Join(tempDir, "dir600", "subdir", "file.txt")

		err := WriteSecureFile(nestedPath, []byte("test"), "0o600", false)
		assert.NilError(t, err)

		dirInfo, err := os.Stat(filepath.Join(tempDir, "dir600"))
		assert.NilError(t, err)
		assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())
	})

	t.Run("directory mode derived from file mode 0o644", func(t *testing.T) {
		nestedPath := filepath.Join(tempDir, "dir644", "subdir", "file.txt")

		err := WriteSecureFile(nestedPath, []byte("test"), "0o644", false)
		assert.NilError(t, err)

		dirInfo, err := os.Stat(filepath.Join(tempDir, "dir644"))
		assert.NilError(t, err)
		assert.Equal(t, os.FileMode(0o744), dirInfo.Mode().Perm())
	})

	t.Run("directory mode derived from file mode 0o400", func(t *testing.T) {
		nestedPath := filepath.Join(tempDir, "dir400", "file.txt")

		err := WriteSecureFile(nestedPath, []byte("test"), "0o400", false)
		assert.NilError(t, err)

		dirInfo, err := os.Stat(filepath.Join(tempDir, "dir400"))
		assert.NilError(t, err)
		assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())
	})
}

func TestFileExists(t *testing.T) {
	t.Run("ExistingFile", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")

		err := os.WriteFile(testFile, []byte("test"), 0644)
		assert.NilError(t, err)

		assert.Assert(t, FileExists(testFile))
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		result := FileExists("/non/existent/file")
		assert.Assert(t, !result)
	})
}

func TestCopyFile(t *testing.T) {
	t.Run("SuccessfulCopy", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceFile := filepath.Join(tempDir, "source.txt")
		destFile := filepath.Join(tempDir, "dest.txt")
		content := "test content"

		err := os.WriteFile(sourceFile, []byte(content), 0644)
		assert.NilError(t, err)

		err = CopyFile(sourceFile, destFile)
		assert.NilError(t, err)

		destContent, err := os.ReadFile(destFile)
		assert.NilError(t, err)
		assert.Equal(t, string(destContent), content)
	})

	t.Run("NonExistentSource", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceFile := filepath.Join(tempDir, "nonexistent.txt")
		destFile := filepath.Join(tempDir, "dest.txt")

		err := CopyFile(sourceFile, destFile)
		assert.ErrorContains(t, err, "failed to stat source file")
	})

	t.Run("DirectoryAsSource", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceDir := filepath.Join(tempDir, "sourcedir")
		destFile := filepath.Join(tempDir, "dest.txt")

		err := os.Mkdir(sourceDir, 0755)
		assert.NilError(t, err)

		err = CopyFile(sourceDir, destFile)
		assert.ErrorContains(t, err, "is not a regular file")
	})
}

func TestParseFileMode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    os.FileMode
		wantErr string
	}{
		{"EmptyStringReturnsDefault", "", DefaultFileMode, ""},
		{"OctalNotation0o600", "0o600", 0o600, ""},
		{"OctalNotation0O644", "0O644", 0o644, ""},
		{"DecimalNotation384", "384", 0o600, ""},
		{"DecimalNotation420", "420", 0o644, ""},
		{"InvalidFormat", "abc", 0, "invalid file mode"},
		{"ExceedsMaxMode", "0o1000", 0, "exceeds 0o777"},
		{"WhitespaceHandling", "  0o600  ", 0o600, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, err := ParseFileMode(tt.input)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			assert.NilError(t, err)
			assert.Equal(t, mode, tt.want)
		})
	}
}
