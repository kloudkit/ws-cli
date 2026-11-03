package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/cmd/clitest"

	"gotest.tools/v3/assert"
)

func run(t *testing.T, args ...string) (string, error) {
	return clitest.Run(TemplateCmd, "", args...)
}

func seedHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	source := filepath.Join(home, ".config", "ruff", "ruff.toml")
	assert.NilError(t, os.MkdirAll(filepath.Dir(source), 0o755))
	assert.NilError(t, os.WriteFile(source, []byte("line-length = 88\n"), 0o644))
	t.Setenv("HOME", home)

	project := filepath.Join(home, "project")
	assert.NilError(t, os.MkdirAll(project, 0o755))
	t.Chdir(project)

	return project
}

func assertHas(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		assert.Assert(t, strings.Contains(out, want), "want %q in %q", want, out)
	}
}

func TestTemplateList(t *testing.T) {
	project := seedHome(t)
	assert.NilError(t, os.WriteFile(filepath.Join(project, ".ruff.toml"), nil, 0o644))

	out, err := run(t, "list")
	assert.NilError(t, err)
	assertHas(t, out, "Templates Available", "(4)", "ansible", "markdownlint", "yamllint", "~/.config/ruff/ruff.toml (applied)", "template apply <name>")
	assert.Equal(t, strings.Count(out, "(applied)"), 1)
}

func TestTemplateShowAndApply(t *testing.T) {
	project := seedHome(t)

	out, err := run(t, "show", "ruff")
	assert.NilError(t, err)
	assertHas(t, out, "Template ruff", "line-length = 88")

	_, err = run(t, "show", "nope")
	assert.ErrorContains(t, err, "template 'nope' not found")

	out, err = run(t, "apply", "ruff", "--path", project)
	assert.NilError(t, err)
	assertHas(t, out, "✓ Template applied", project+"/.ruff.toml", "template show --local ruff")

	content, err := os.ReadFile(filepath.Join(project, ".ruff.toml"))
	assert.NilError(t, err)
	assert.Equal(t, string(content), "line-length = 88\n")

	out, err = run(t, "show", "ruff", "--local")
	assert.NilError(t, err)
	assertHas(t, out, "Local Template ruff", "line-length = 88")

	_, err = run(t, "apply", "ruff", "--path", project)
	assert.ErrorContains(t, err, "file already exists")
}
