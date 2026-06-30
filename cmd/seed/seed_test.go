package seed

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
)

func resetCommandFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flag.Value.Set(flag.DefValue)
		flag.Changed = false
	})

	for _, c := range cmd.Commands() {
		resetCommandFlags(c)
	}
}

func run(t *testing.T, args ...string) string {
	t.Helper()
	resetCommandFlags(SeedCmd)

	buffer := new(bytes.Buffer)
	SeedCmd.SetOut(buffer)
	SeedCmd.SetErr(buffer)
	SeedCmd.SetArgs(args)

	assert.NilError(t, SeedCmd.Execute())

	return buffer.String()
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
}
