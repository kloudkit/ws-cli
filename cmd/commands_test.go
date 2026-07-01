package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kloudkit/ws-cli/cmd"
	"github.com/kloudkit/ws-cli/internals/docs"
	"gotest.tools/v3/assert"
)

func TestCommandsManifestMatchesCommittedFile(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("..", "commands.yaml"))
	assert.NilError(t, err)

	got, err := docs.Serialize(cmd.RootCmd())
	assert.NilError(t, err)

	assert.Equal(t, string(got), string(want))
}

func TestSerializeIsDeterministic(t *testing.T) {
	first, err := docs.Serialize(cmd.RootCmd())
	assert.NilError(t, err)

	second, err := docs.Serialize(cmd.RootCmd())
	assert.NilError(t, err)

	assert.Equal(t, string(first), string(second))
}
