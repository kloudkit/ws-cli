package path

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestAppendSegments(t *testing.T) {
	assert.Equal(t, AppendSegments("/", "path"), "/path")
	assert.Equal(t, AppendSegments("/home/", "/sub", "path/"), "/home/sub/path")
}

func TestHome(t *testing.T) {
	t.Setenv("HOME", "")
	assert.Equal(t, GetHomeDirectory(), "/home/kloud")

	t.Setenv("HOME", "/home/user")
	assert.Equal(t, GetHomeDirectory(), "/home/user")
	assert.Equal(t, ResolveConfigPath("/etc/config"), "/etc/config")
	assert.Equal(t, ResolveConfigPath(".config/app/config"), "/home/user/.config/app/config")
	assert.Equal(t, ShortenHomePath("/home/user/.config"), "~/.config")
	assert.Equal(t, ShortenHomePath("/etc/config"), "/etc/config")
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
		assert.Equal(t, Expand(c.in), c.want, c.in)
	}
}
