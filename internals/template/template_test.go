package template

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestGetTemplate(t *testing.T) {
	config, exists := GetTemplate("markdownlint")
	assert.Assert(t, exists, "markdownlint template should exist")
	assert.Equal(t, config.SourcePath, ".config/markdownlint/config")
	assert.Equal(t, config.OutputName, ".markdownlint.json")

	_, exists = GetTemplate("nonexistent")
	assert.Assert(t, !exists, "nonexistent template should not exist")
}

func TestGetTemplateNames(t *testing.T) {
	names := GetTemplateNames()
	expectedNames := []string{"ansible", "markdownlint", "ruff", "yamllint"}

	assert.Equal(t, len(names), len(expectedNames))

	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	for _, expected := range expectedNames {
		assert.Assert(t, nameSet[expected], "expected template name '%s' not found", expected)
	}
}

func TestApplyTemplate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "template_test")
	assert.NilError(t, err)
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, ".config", "markdownlint")
	err = os.MkdirAll(sourceDir, 0755)
	assert.NilError(t, err)

	sourceFile := filepath.Join(sourceDir, "config")
	err = os.WriteFile(sourceFile, []byte(`{"line-length": false}`), 0644)
	assert.NilError(t, err)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	targetDir := filepath.Join(tempDir, "project")
	err = os.MkdirAll(targetDir, 0755)
	assert.NilError(t, err)

	err = ApplyTemplate("markdownlint", targetDir, false)
	assert.NilError(t, err)

	destFile := filepath.Join(targetDir, ".markdownlint.json")
	_, err = os.Stat(destFile)
	assert.NilError(t, err, "destination file was not created")

	content, err := os.ReadFile(destFile)
	assert.NilError(t, err)

	expected := `{"line-length": false}`
	assert.Equal(t, string(content), expected)
}

func TestApplyTemplateWithForce(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "template_test")
	assert.NilError(t, err)
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, ".config", "ruff")
	err = os.MkdirAll(sourceDir, 0755)
	assert.NilError(t, err)

	sourceFile := filepath.Join(sourceDir, "ruff.toml")
	err = os.WriteFile(sourceFile, []byte(`line-length = 88`), 0644)
	assert.NilError(t, err)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

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
}

func TestShowTemplate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "template_test")
	assert.NilError(t, err)
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, ".config", "yamllint")
	err = os.MkdirAll(sourceDir, 0755)
	assert.NilError(t, err)

	sourceFile := filepath.Join(sourceDir, "config")
	expectedContent := `extends: default
rules:
  line-length:
    max: 120`
	err = os.WriteFile(sourceFile, []byte(expectedContent), 0644)
	assert.NilError(t, err)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	content, err := ShowTemplate("yamllint", false)
	assert.NilError(t, err)

	assert.Equal(t, content, expectedContent)
}
