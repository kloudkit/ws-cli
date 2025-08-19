package path_test

import (
	"os"
	"testing"

	"github.com/kloudkit/ws-cli/internals/path"
	"gotest.tools/v3/assert"
)

func TestAppendSegments(t *testing.T) {
	t.Run("AdditionalSegments", func(t *testing.T) {
		assert.Equal(t, "/path", path.AppendSegments("/", "path"))
	})

	t.Run("NormalizeAdditionalSegments", func(t *testing.T) {
		assert.Equal(t, "/home/sub/path", path.AppendSegments("/home/", "/sub", "path/"))
	})
}

func TestGetHomeDirectory(t *testing.T) {
	t.Run("WithEnv", func(t *testing.T) {
		t.Setenv("HOME", "/app")

		assert.Equal(t, "/app", path.GetHomeDirectory())
	})

	t.Run("WithoutEnv", func(t *testing.T) {
		os.Unsetenv("HOME")

		assert.Equal(t, "/home/kloud", path.GetHomeDirectory())
	})
}
