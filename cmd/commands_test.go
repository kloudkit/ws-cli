package cmd_test

import (
	"cmp"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/cmd"
	"github.com/kloudkit/ws-cli/cmd/editor"
	"github.com/kloudkit/ws-cli/internals/docs"
	"github.com/kloudkit/ws-cli/internals/mcp"
	"github.com/kloudkit/ws-cli/internals/testutil"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
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

func TestReapArgsNameTheHiddenReapCommand(t *testing.T) {
	found, rest, err := cmd.RootCmd().Find(editor.ReapArgs)

	assert.NilError(t, err)
	assert.Equal(t, len(rest), 0)
	assert.Equal(t, found.CommandPath(), "ws-cli editor reap-download")
	assert.Assert(t, found.Hidden)
}

var flagMention = regexp.MustCompile(`--[a-z][a-z0-9-]*`)

func TestMcpAnnotationsAreWellFormed(t *testing.T) {
	testutil.FakeWorkspace(t)

	tools := 0

	var check func(c *cobra.Command)
	check = func(c *cobra.Command) {
		spec, annotated, err := mcp.ParseSpec(c.Annotations)
		assert.NilError(t, err, c.CommandPath())

		if annotated {
			tools++

			assert.Assert(t, c.Runnable(), "%q is a tool but cannot run", c.CommandPath())
			assert.Assert(t, !c.Hidden, "%q is a tool but hidden", c.CommandPath())

			for _, name := range spec.Flags {
				assert.Assert(t, c.Flag(name) != nil, "%q exposes the unknown flag %q", c.CommandPath(), name)
			}

			for _, flag := range flagMention.FindAllString(cmp.Or(c.Long, c.Short), -1) {
				assert.Assert(t, slices.Contains(spec.Flags, strings.TrimPrefix(flag, "--")),
					"%q describes %s, which its tool does not expose: expose it or reword the help", c.CommandPath(), flag)
			}
		}

		for _, child := range c.Commands() {
			check(child)
		}
	}

	check(cmd.RootCmd())
	assert.Assert(t, tools > 0)

	built, _, err := mcp.BuildTools(cmd.RootCmd(), mcp.Settings{AllowWrite: true})
	assert.NilError(t, err)
	assert.Assert(t, len(built) > 0 && len(built) <= tools)
}
