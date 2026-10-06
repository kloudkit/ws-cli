package path

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func TestAppendSegments(t *testing.T) {
	t.Run("AdditionalSegments", func(t *testing.T) {
		assert.Equal(t, "/path", AppendSegments("/", "path"))
	})

	t.Run("NormalizeAdditionalSegments", func(t *testing.T) {
		assert.Equal(t, "/home/sub/path", AppendSegments("/home/", "/sub", "path/"))
	})
}

func TestGetHomeDirectory(t *testing.T) {
	t.Run("WithEnv", func(t *testing.T) {
		t.Setenv("HOME", "/app")

		assert.Equal(t, "/app", GetHomeDirectory())
	})

	t.Run("WithoutEnv", func(t *testing.T) {
		t.Setenv("HOME", "")

		assert.Equal(t, "/home/kloud", GetHomeDirectory())
	})
}

func TestResolveConfigPath(t *testing.T) {
	t.Run("AbsolutePath", func(t *testing.T) {
		result := ResolveConfigPath("/etc/config")
		assert.Equal(t, "/etc/config", result)
	})

	t.Run("RelativePath", func(t *testing.T) {
		t.Setenv("HOME", "/home/user")
		result := ResolveConfigPath(".config/app/config")
		assert.Equal(t, "/home/user/.config/app/config", result)
	})
}

func TestExpand(t *testing.T) {
	t.Setenv("HOME", "/home/testuser")
	t.Setenv("TEST_DIR", "/var/test")

	cases := []struct{ in, want string }{
		{"/etc/config", "/etc/config"},
		{"~/.config", "/home/testuser/.config"},
		{"~", "/home/testuser"},
		{"$TEST_DIR/config", "/var/test/config"},
		{"/etc//config/../config/app", "/etc/config/app"},
		{"~root/x", "~root/x"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, Expand(c.in), c.in)
	}
}

func TestGetCurrentWorkingDirectory(t *testing.T) {
	t.Run("WithoutSegments", func(t *testing.T) {
		result, err := GetCurrentWorkingDirectory()
		assert.NilError(t, err)

		cwd, err := os.Getwd()
		assert.NilError(t, err)
		assert.Equal(t, result, cwd)
	})

	t.Run("WithSegments", func(t *testing.T) {
		result, err := GetCurrentWorkingDirectory("sub", "path")
		assert.NilError(t, err)

		cwd, err := os.Getwd()
		assert.NilError(t, err)
		expected := AppendSegments(cwd, "sub", "path")
		assert.Equal(t, result, expected)
	})
}
