package template

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestLookup(t *testing.T) {
	t.Run("ExistingTemplate", func(t *testing.T) {
		config, err := lookup("markdownlint")
		assert.NilError(t, err)
		assert.Equal(t, config.SourcePath, ".config/markdownlint/config")
		assert.Equal(t, config.OutputName, ".markdownlint.json")
	})

	t.Run("NonExistentTemplate", func(t *testing.T) {
		_, err := lookup("nonexistent")
		assert.ErrorContains(t, err, "template 'nonexistent' not found")
	})
}

func TestGetTemplateNames(t *testing.T) {
	t.Run("ReturnsAllTemplates", func(t *testing.T) {
		assert.DeepEqual(t, GetTemplateNames(), []string{"ansible", "markdownlint", "ruff", "yamllint"})
	})
}

func seedTemplate(t *testing.T, rel, content string) string {
	t.Helper()

	home := t.TempDir()
	source := filepath.Join(home, rel)
	assert.NilError(t, os.MkdirAll(filepath.Dir(source), 0o755))
	assert.NilError(t, os.WriteFile(source, []byte(content), 0o644))
	t.Setenv("HOME", home)

	return home
}

func TestApplyTemplate(t *testing.T) {
	t.Run("CopiesTemplateToTarget", func(t *testing.T) {
		tempDir := seedTemplate(t, ".config/markdownlint/config", `{"line-length": false}`)

		targetDir := filepath.Join(tempDir, "project")
		err := os.MkdirAll(targetDir, 0755)
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
		tempDir := seedTemplate(t, ".config/ruff/ruff.toml", `line-length = 88`)

		targetDir := filepath.Join(tempDir, "project")
		err := os.MkdirAll(targetDir, 0755)
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
		expectedContent := `extends: default
rules:
  line-length:
    max: 120`
		seedTemplate(t, ".config/yamllint/config", expectedContent)

		content, err := ShowTemplate("yamllint", false)
		assert.NilError(t, err)

		assert.Equal(t, content, expectedContent)
	})
}
