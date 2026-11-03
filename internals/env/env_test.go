package env

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestString(t *testing.T) {
	t.Setenv("FOO", "")
	assert.Equal(t, String("FOO", "bar"), "bar")
	assert.Equal(t, String("FOO"), "")

	t.Setenv("FOO", "baz")
	assert.Equal(t, String("FOO", "bar"), "baz")
	assert.Equal(t, GetAll()["FOO"], "baz")

	t.Setenv("HOME", "")
	assert.Equal(t, Home(), "/home/kloud")
}

func TestIsSSHSession(t *testing.T) {
	keys := []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"}

	for _, key := range keys {
		t.Setenv(key, "")
	}
	assert.Assert(t, !IsSSHSession())

	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "set")
			assert.Assert(t, IsSSHSession())
		})
	}
}
