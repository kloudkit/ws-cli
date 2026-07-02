package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kloudkit/ws-cli/cmd"
	"github.com/kloudkit/ws-cli/internals/docs"
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
