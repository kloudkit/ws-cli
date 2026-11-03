package seed

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/cmd/clitest"

	"gotest.tools/v3/assert"
)

func run(t *testing.T, args ...string) (string, error) {
	return clitest.Run(SeedCmd, "", args...)
}

func source(t *testing.T, manifest string, files ...string) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("WS__INTERNAL_ENV_REFERENCE", filepath.Join(t.TempDir(), "absent.yaml"))

	dir := t.TempDir()
	if manifest != "" {
		assert.NilError(t, os.WriteFile(filepath.Join(dir, ".seed.yaml"), []byte(manifest), 0o644))
	}

	for _, file := range files {
		assert.NilError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, file)), 0o755))
		assert.NilError(t, os.WriteFile(filepath.Join(dir, file), []byte(file), 0o644))
	}

	return dir
}

func TestApply(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "out.txt")
	src := source(t, fmt.Sprintf("version: v1\nseeds:\n  %s:\n    mode: \"0o644\"\n    content: \"cli\\n\"\n", dest))

	output, err := run(t, "apply", "--source", src)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(output, "Seeded ["+dest+"]"), output)

	got, err := os.ReadFile(dest)
	assert.NilError(t, err)
	assert.Equal(t, string(got), "cli\n")
}

func TestList(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "out.txt")
	output, err := run(t, "ls", "--source", source(t, fmt.Sprintf("version: v1\nseeds:\n  %s:\n    secret: true\n", dest)))
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(output, dest) && strings.Contains(output, "secret"), output)

	output, err = run(t, "ls", "--source", source(t, "", "etc/a.conf", "etc/b.conf"))
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(output, "(2)") && strings.Contains(output, "/etc/a.conf"), output)
}

func TestSourceEdges(t *testing.T) {
	for _, verb := range []string{"ls", "apply"} {
		t.Run(verb, func(t *testing.T) {
			empty := source(t, "")
			output, err := run(t, verb, "--source", empty)
			assert.NilError(t, err)
			assert.Assert(t, strings.Contains(output, "No seed entries in ["+empty+"]"), output)

			missing := filepath.Join(empty, "missing")
			_, err = run(t, verb, "--source", missing)
			assert.ErrorContains(t, err, "seed source ["+missing+"] not found")
		})
	}
}
