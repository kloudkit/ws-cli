package get

import (
	"bytes"
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func TestHome(t *testing.T) {
	t.Run("WithEnv", func(t *testing.T) {
		t.Setenv("WS_ROOT", "/app")

		assertOutputEquals(t, []string{"home"}, "/app\n")
	})

	t.Run("WithoutEnv", func(t *testing.T) {
		os.Unsetenv("WS_ROOT")

		assertOutputEquals(t, []string{"home"}, "/workspace\n")
	})
}

func TestSettings(t *testing.T) {
	t.Run("WithEnv", func(t *testing.T) {
		t.Setenv("HOME", "/app")

		assertOutputEquals(t, []string{"settings"}, "/app/.local/share/code-server/User/settings.json\n")
	})

	t.Run("WithoutEnv", func(t *testing.T) {
		os.Unsetenv("HOME")

		assertOutputEquals(t, []string{"settings"}, "/home/kloud/.local/share/code-server/User/settings.json\n")
	})
}

func assertOutputEquals(t *testing.T, args []string, expected string) {
	buffer := new(bytes.Buffer)
	cmd := GetCmd

	cmd.SetOut(buffer)
	cmd.SetArgs(args)
	cmd.Execute()

	assert.Equal(t, expected, buffer.String())
}
