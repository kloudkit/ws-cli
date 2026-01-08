package template

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestGetTemplate(t *testing.T) {
	t.Run("ExistingTemplate", func(t *testing.T) {
		config, exists := GetTemplate("markdownlint")
		assert.Assert(t, exists)
		assert.Equal(t, config.SourcePath, ".config/markdownlint/config")
		assert.Equal(t, config.OutputName, ".markdownlint.json")
	})

	t.Run("NonExistentTemplate", func(t *testing.T) {
		_, exists := GetTemplate("nonexistent")
		assert.Assert(t, !exists)
	})
}

func TestGetTemplateNames(t *testing.T) {
	t.Run("ReturnsAllTemplates", func(t *testing.T) {
		names := GetTemplateNames()
		expectedNames := []string{"ansible", "markdownlint", "ruff", "yamllint"}

		assert.Equal(t, len(names), len(expectedNames))

		nameSet := make(map[string]bool)
		for _, name := range names {
			nameSet[name] = true
		}

		for _, expected := range expectedNames {
			assert.Assert(t, nameSet[expected])
		}
	})
}

func TestApplyTemplate(t *testing.T) {
	t.Run("CopiesTemplateToTarget", func(t *testing.T) {
		tempDir := t.TempDir()

		sourceDir := filepath.Join(tempDir, ".config", "markdownlint")
		err := os.MkdirAll(sourceDir, 0755)
		assert.NilError(t, err)

		sourceFile := filepath.Join(sourceDir, "config")
		err = os.WriteFile(sourceFile, []byte(`{"line-length": false}`), 0644)
		assert.NilError(t, err)

		t.Setenv("HOME", tempDir)

		targetDir := filepath.Join(tempDir, "project")
		err = os.MkdirAll(targetDir, 0755)
		assert.NilError(t, err)

		err = ApplyTemplate("markdownlint", targetDir, false)
		assert.NilError(t, err)

		destFile := filepath.Join(targetDir, ".markdownlint.json")
		_, err = os.Stat(destFile)
		assert.NilError(t, err)

		content, err := os.ReadFile(destFile)
		assert.NilError(t, err)

		expected := `{"line-length": false}`
		assert.Equal(t, string(content), expected)
	})

	t.Run("WithForceOverwritesExisting", func(t *testing.T) {
		tempDir := t.TempDir()

		sourceDir := filepath.Join(tempDir, ".config", "ruff")
		err := os.MkdirAll(sourceDir, 0755)
		assert.NilError(t, err)

		sourceFile := filepath.Join(sourceDir, "ruff.toml")
		err = os.WriteFile(sourceFile, []byte(`line-length = 88`), 0644)
		assert.NilError(t, err)

		t.Setenv("HOME", tempDir)

		targetDir := filepath.Join(tempDir, "project")
		err = os.MkdirAll(targetDir, 0755)
		assert.NilError(t, err)

		destFile := filepath.Join(targetDir, ".ruff.toml")
		err = os.WriteFile(destFile, []byte("existing content"), 0644)
		assert.NilError(t, err)

		err = ApplyTemplate("ruff", targetDir, false)
		assert.ErrorContains(t, err, "file already exists")

		err = ApplyTemplate("ruff", targetDir, true)
		assert.NilError(t, err)

		content, err := os.ReadFile(destFile)
		assert.NilError(t, err)

		expected := `line-length = 88`
		assert.Equal(t, string(content), expected)
	})
}

func TestShowTemplate(t *testing.T) {
	t.Run("ReturnsTemplateContent", func(t *testing.T) {
		tempDir := t.TempDir()

		sourceDir := filepath.Join(tempDir, ".config", "yamllint")
		err := os.MkdirAll(sourceDir, 0755)
		assert.NilError(t, err)

		sourceFile := filepath.Join(sourceDir, "config")
		expectedContent := `extends: default
rules:
  line-length:
    max: 120`
		err = os.WriteFile(sourceFile, []byte(expectedContent), 0644)
		assert.NilError(t, err)

		t.Setenv("HOME", tempDir)

		content, err := ShowTemplate("yamllint", false)
		assert.NilError(t, err)

		assert.Equal(t, content, expectedContent)
	})
}
