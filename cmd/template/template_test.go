package template

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
)

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	testutil.ResetCommandFlags(TemplateCmd)

	buffer := new(bytes.Buffer)
	TemplateCmd.SetOut(buffer)
	TemplateCmd.SetErr(buffer)
	TemplateCmd.SetArgs(args)

	err := TemplateCmd.Execute()

	return testutil.StripAnsi(buffer.String()), err
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

func TestTemplateList(t *testing.T) {
	project := seedHome(t)
	assert.NilError(t, os.WriteFile(filepath.Join(project, ".ruff.toml"), nil, 0o644))

	out, err := run(t, "list")
	assert.NilError(t, err)

	assert.Assert(t, strings.Contains(strings.ToLower(out), "templates available"))
	assert.Assert(t, strings.Contains(out, "(4)"))

	for _, name := range []string{"ansible", "markdownlint", "ruff", "yamllint"} {
		assert.Assert(t, strings.Contains(out, name), "missing %s in %q", name, out)
	}

	assert.Assert(t, strings.Contains(out, "~/.config/ruff/ruff.toml"))
	assert.Equal(t, strings.Count(out, "(applied)"), 1)
	assert.Assert(t, strings.Contains(out, "  • ruff         — ~/.config/ruff/ruff.toml (applied)\n"), "got %q", out)
	assert.Assert(t, strings.Contains(out, "template apply <name>"))
}

func TestTemplateShow(t *testing.T) {
	seedHome(t)

	out, err := run(t, "show", "ruff")
	assert.NilError(t, err)

	assert.Assert(t, strings.Contains(out, "Template ruff"))
	assert.Assert(t, strings.Contains(out, "\n    line-length = 88\n"), "got %q", out)
}

func TestTemplateShowUnknown(t *testing.T) {
	seedHome(t)

	_, err := run(t, "show", "nope")
	assert.ErrorContains(t, err, "template 'nope' not found")
}

func TestTemplateApply(t *testing.T) {
	project := seedHome(t)

	out, err := run(t, "apply", "ruff", "--path", project)
	assert.NilError(t, err)

	content, err := os.ReadFile(filepath.Join(project, ".ruff.toml"))
	assert.NilError(t, err)
	assert.Equal(t, string(content), "line-length = 88\n")

	assert.Assert(t, strings.Contains(out, "✓ Template applied"))
	assert.Assert(t, strings.Contains(out, project+"/.ruff.toml"))
	assert.Assert(t, strings.Contains(out, "template show --local ruff"))
}

func TestTemplateApplyRefusesExisting(t *testing.T) {
	project := seedHome(t)
	assert.NilError(t, os.WriteFile(filepath.Join(project, ".ruff.toml"), []byte("mine"), 0o644))

	_, err := run(t, "apply", "ruff", "--path", project)
	assert.ErrorContains(t, err, "file already exists")
}
