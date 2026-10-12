package seed

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
)

func run(t *testing.T, args ...string) string {
	t.Helper()
	testutil.ResetCommandFlags(SeedCmd)

	buffer := new(bytes.Buffer)
	SeedCmd.SetOut(buffer)
	SeedCmd.SetErr(buffer)
	SeedCmd.SetArgs(args)

	assert.NilError(t, SeedCmd.Execute())

	return buffer.String()
}

func runErr(t *testing.T, args ...string) error {
	t.Helper()
	testutil.ResetCommandFlags(SeedCmd)

	buffer := new(bytes.Buffer)
	SeedCmd.SetOut(buffer)
	SeedCmd.SetErr(buffer)
	SeedCmd.SetArgs(args)

	return SeedCmd.Execute()
}

func TestSeedCommand(t *testing.T) {
	t.Run("Apply", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("WS__INTERNAL_ENV_REFERENCE", filepath.Join(t.TempDir(), "absent.yaml"))

		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "out.txt")

		manifest := fmt.Sprintf("version: v1\nseeds:\n  %s:\n    mode: \"0o644\"\n    content: \"cli\\n\"\n", dest)
		assert.NilError(t, os.WriteFile(filepath.Join(source, ".seed.yaml"), []byte(manifest), 0o644))

		output := run(t, "apply", "--source", source)

		got, err := os.ReadFile(dest)
		assert.NilError(t, err)
		assert.Equal(t, string(got), "cli\n")
		assert.Assert(t, strings.Contains(output, "Seeded ["+dest+"]"))
	})

	t.Run("List", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("WS__INTERNAL_ENV_REFERENCE", filepath.Join(t.TempDir(), "absent.yaml"))

		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "out.txt")

		manifest := fmt.Sprintf("version: v1\nseeds:\n  %s:\n    secret: true\n", dest)
		assert.NilError(t, os.WriteFile(filepath.Join(source, ".seed.yaml"), []byte(manifest), 0o644))

		output := run(t, "ls", "--source", source)

		assert.Assert(t, strings.Contains(output, dest))
		assert.Assert(t, strings.Contains(output, "secret"))
	})
	for _, verb := range []string{"ls", "apply"} {
		t.Run("EmptySource_"+verb, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("WS__INTERNAL_ENV_REFERENCE", filepath.Join(t.TempDir(), "absent.yaml"))

			source := t.TempDir()

			output := testutil.StripAnsi(run(t, verb, "--source", source))

			assert.Assert(t, strings.Contains(output, "No seed entries in ["+source+"]"))
		})

		t.Run("MissingExplicitSource_"+verb, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("WS__INTERNAL_ENV_REFERENCE", filepath.Join(t.TempDir(), "absent.yaml"))

			source := filepath.Join(t.TempDir(), "missing")

			err := runErr(t, verb, "--source", source)

			assert.ErrorContains(t, err, "seed source ["+source+"] not found")
		})
	}

	t.Run("ListTitleCountsEntries", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("WS__INTERNAL_ENV_REFERENCE", filepath.Join(t.TempDir(), "absent.yaml"))

		source := t.TempDir()
		assert.NilError(t, os.MkdirAll(filepath.Join(source, "etc"), 0o755))
		assert.NilError(t, os.WriteFile(filepath.Join(source, "etc", "a.conf"), []byte("a"), 0o644))
		assert.NilError(t, os.WriteFile(filepath.Join(source, "etc", "b.conf"), []byte("b"), 0o644))

		output := testutil.StripAnsi(run(t, "ls", "--source", source))

		assert.Assert(t, strings.Contains(output, "(2)"))
		assert.Assert(t, strings.Contains(output, "/etc/a.conf"))
	})
}
