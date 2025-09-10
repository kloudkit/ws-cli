package show

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestPathHome(t *testing.T) {
	t.Run("WithEnv", func(t *testing.T) {
		t.Setenv("WS_SERVER_ROOT", "/app")

		assertOutputContains(t, []string{"path", "home"}, "/app")
	})

	t.Run("WithoutEnv", func(t *testing.T) {
		os.Unsetenv("WS_SERVER_ROOT")

		assertOutputContains(t, []string{"path", "home"}, "/workspace")
	})
}

func TestPathVscode(t *testing.T) {
	t.Run("WithEnv", func(t *testing.T) {
		t.Setenv("HOME", "/app")

		assertOutputContains(t, []string{"path", "vscode-settings"}, "/app/.local/share/workspace/User/settings.json")
	})

	t.Run("WithoutEnv", func(t *testing.T) {
		os.Unsetenv("HOME")

		assertOutputContains(t, []string{"path", "vscode-settings"}, "/home/kloud/.local/share/workspace/User/settings.json")
	})

	t.Run("WorkspaceSettings", func(t *testing.T) {
		assertOutputContains(t, []string{"path", "vscode-settings", "--workspace"}, "/workspace/.vscode/settings.json")
	})
}

func assertOutputContains(t *testing.T, args []string, expected string) {
	buffer := new(bytes.Buffer)
	cmd := ShowCmd

	cmd.SetOut(buffer)
	cmd.SetArgs(args)
	cmd.Execute()

	output := buffer.String()

	assert.Assert(t, strings.Contains(output, expected))
}
