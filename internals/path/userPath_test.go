package path_test

import (
	"os"
	"testing"

	"github.com/kloudkit/ws-cli/internals/path"
	"gotest.tools/v3/assert"
)

func TestGetHomeDirectory(t *testing.T) {
	t.Run("WithEnv", func(t *testing.T) {
		t.Setenv("HOME", "/app")

  	assert.Equal(t, "/app", path.GetHomeDirectory())
	})

	t.Run("WithoutEnv", func(t *testing.T) {
    os.Unsetenv("HOME")

    assert.Equal(t, "/home/kloud", path.GetHomeDirectory())
	})

  t.Run("AdditionalSegments", func(t *testing.T) {
    os.Unsetenv("HOME")

    assert.Equal(t, "/home/kloud/sub/path", path.GetHomeDirectory("sub", "path"))
	})

  t.Run("NormalizeAdditionalSegments", func(t *testing.T) {
    os.Unsetenv("HOME")

    assert.Equal(t, "/home/kloud/sub/path", path.GetHomeDirectory("/sub", "path/"))
	})
}
