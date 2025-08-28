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

		output := assertOutputContains(t, []string{"path", "home"}, "/app")
		assert.Assert(t, strings.Contains(output, "/app"))
	})

	t.Run("WithoutEnv", func(t *testing.T) {
		os.Unsetenv("WS_SERVER_ROOT")

		output := assertOutputContains(t, []string{"path", "home"}, "/workspace")
		assert.Assert(t, strings.Contains(output, "/workspace"))
	})
}

func TestPathVscode(t *testing.T) {
	t.Run("WithEnv", func(t *testing.T) {
		t.Setenv("HOME", "/app")

		output := assertOutputContains(t, []string{"path", "vscode-settings"}, "/app/.local/share/code-server/User/settings.json")
		assert.Assert(t, strings.Contains(output, "/app/.local/share/code-server/User/settings.json"))
	})

	t.Run("WithoutEnv", func(t *testing.T) {
		os.Unsetenv("HOME")

		output := assertOutputContains(t, []string{"path", "vscode-settings"}, "/home/kloud/.local/share/code-server/User/settings.json")
		assert.Assert(t, strings.Contains(output, "/home/kloud/.local/share/code-server/User/settings.json"))
	})

	t.Run("WorkspaceSettings", func(t *testing.T) {
		output := assertOutputContains(t, []string{"path", "vscode-settings", "--workspace"}, "/workspace/.vscode/settings.json")
		assert.Assert(t, strings.Contains(output, "/workspace/.vscode/settings.json"))
	})
}

func assertOutputContains(t *testing.T, args []string, expected string) string {
	buffer := new(bytes.Buffer)
	cmd := ShowCmd

	cmd.SetOut(buffer)
	cmd.SetArgs(args)
	cmd.Execute()

	output := buffer.String()
	assert.Assert(t, strings.Contains(output, expected))
	return output
}
