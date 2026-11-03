package template

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestLookup(t *testing.T) {
	config, err := lookup("markdownlint")
	assert.NilError(t, err)
	assert.DeepEqual(t, config, Config{SourcePath: ".config/markdownlint/config", OutputName: ".markdownlint.json"})

	_, err = lookup("nonexistent")
	assert.ErrorContains(t, err, "template 'nonexistent' not found")

	assert.DeepEqual(t, GetTemplateNames(), []string{"ansible", "markdownlint", "ruff", "yamllint"})
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
	target := filepath.Join(seedTemplate(t, ".config/ruff/ruff.toml", "line-length = 88"), "project")
	assert.NilError(t, os.MkdirAll(target, 0o755))
	dest := filepath.Join(target, ".ruff.toml")

	assert.NilError(t, ApplyTemplate("ruff", target, false))
	content, err := os.ReadFile(dest)
	assert.NilError(t, err)
	assert.Equal(t, string(content), "line-length = 88")

	assert.NilError(t, os.WriteFile(dest, []byte("existing content"), 0o644))
	assert.ErrorContains(t, ApplyTemplate("ruff", target, false), "file already exists")

	assert.NilError(t, ApplyTemplate("ruff", target, true))
	content, err = os.ReadFile(dest)
	assert.NilError(t, err)
	assert.Equal(t, string(content), "line-length = 88")

	assert.ErrorContains(t, ApplyTemplate("yamllint", target, false), "template source file not found")
}

func TestShowTemplate(t *testing.T) {
	const body = "extends: default\nrules:\n  line-length:\n    max: 120"
	seedTemplate(t, ".config/yamllint/config", body)

	content, err := ShowTemplate("yamllint", false)
	assert.NilError(t, err)
	assert.Equal(t, content, body)

	t.Chdir(t.TempDir())
	assert.NilError(t, os.WriteFile(".yamllint", []byte("local"), 0o644))

	content, err = ShowTemplate("yamllint", true)
	assert.NilError(t, err)
	assert.Equal(t, content, "local")
}
