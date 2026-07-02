package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/cmd"
	"github.com/kloudkit/ws-cli/internals/docs"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert"
)

func TestCommandsManifestMatchesCommittedFile(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("..", "commands.yaml"))
	assert.NilError(t, err)

	got, err := docs.Serialize(cmd.RootCmd())
	assert.NilError(t, err)

	assert.Equal(t, string(got), string(want))
}

func TestEveryCommandCarriesVersion(t *testing.T) {
	out, err := docs.Serialize(cmd.RootCmd())
	assert.NilError(t, err)

	var root docs.Command
	assert.NilError(t, yaml.Unmarshal(out, &root))

	var check func(docs.Command)
	check = func(c docs.Command) {
		for _, child := range c.Commands {
			assert.Assert(t, child.Since != "" || child.Deprecated != "",
				"command %q must carry a since or deprecated annotation", child.Name)
			check(child)
		}
	}

	check(root)
}

func TestSerializeIsDeterministic(t *testing.T) {
	first, err := docs.Serialize(cmd.RootCmd())
	assert.NilError(t, err)

	second, err := docs.Serialize(cmd.RootCmd())
	assert.NilError(t, err)

	assert.Equal(t, string(first), string(second))
}

func TestExampleSerializesIntoManifest(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	root.AddCommand(&cobra.Command{
		Use:     "child",
		Example: "ws root child --flag",
		Run:     func(*cobra.Command, []string) {},
	})

	out, err := docs.Serialize(root)
	assert.NilError(t, err)

	var parsed docs.Command
	assert.NilError(t, yaml.Unmarshal(out, &parsed))

	assert.Equal(t, len(parsed.Commands), 1)
	assert.Equal(t, parsed.Commands[0].Example, "ws root child --flag")

	bare := &cobra.Command{Use: "root"}
	bare.AddCommand(&cobra.Command{Use: "child", Run: func(*cobra.Command, []string) {}})

	bareOut, err := docs.Serialize(bare)
	assert.NilError(t, err)

	assert.Assert(t, !strings.Contains(string(bareOut), "example:"),
		"expected no example: key when Example is unset (omitempty)")
}

func TestExamplesAreTrimmedAndNonEmpty(t *testing.T) {
	out, err := docs.Serialize(cmd.RootCmd())
	assert.NilError(t, err)

	var root docs.Command
	assert.NilError(t, yaml.Unmarshal(out, &root))

	found := false

	var check func(docs.Command)
	check = func(c docs.Command) {
		if c.Example != "" {
			found = true
			assert.Equal(t, strings.TrimSpace(c.Example), c.Example,
				"command %q example must not carry leading or trailing whitespace", c.Name)
		}

		for _, child := range c.Commands {
			check(child)
		}
	}

	check(root)

	assert.Assert(t, found, "expected at least one command to carry an example")
}
