package env

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestString(t *testing.T) {
	t.Run("DefaultValue", func(t *testing.T) {
		t.Setenv("FOO", "")

		assert.Equal(t, "bar", String("FOO", "bar"))
	})

	t.Run("EnvValue", func(t *testing.T) {
		t.Setenv("FOO", "baz")

		assert.Equal(t, "baz", String("FOO", "bar"))
	})
}

func TestMustString(t *testing.T) {
	t.Run("PanicWhenMissing", func(t *testing.T) {
		t.Setenv("FOO", "")

		assert.Assert(t, func() (result bool) {
			defer func() {
				result = recover() != nil
			}()

			MustString("FOO")

			return false
		}())
	})

	t.Run("WithFallback", func(t *testing.T) {
		t.Setenv("FOO", "")

		assert.Equal(t, "qux", MustString("FOO", "qux"))
	})
}

func TestIsSSHSession(t *testing.T) {
	clear := func(t *testing.T) {
		for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
			t.Setenv(key, "")
		}
	}

	t.Run("NotSSH", func(t *testing.T) {
		clear(t)

		assert.Assert(t, !IsSSHSession())
	})

	t.Run("SSHConnection", func(t *testing.T) {
		clear(t)
		t.Setenv("SSH_CONNECTION", "1.2.3.4 51000 5.6.7.8 22")

		assert.Assert(t, IsSSHSession())
	})

	t.Run("SSHTTY", func(t *testing.T) {
		clear(t)
		t.Setenv("SSH_TTY", "/dev/pts/0")

		assert.Assert(t, IsSSHSession())
	})
}

func TestGetAll(t *testing.T) {
	t.Run("ReturnsAllEnvVars", func(t *testing.T) {
		t.Setenv("TEST_KEY1", "value1")
		t.Setenv("TEST_KEY2", "value2")

		envMap := GetAll()

		assert.Equal(t, "value1", envMap["TEST_KEY1"])
		assert.Equal(t, "value2", envMap["TEST_KEY2"])
		assert.Assert(t, len(envMap) > 0)
	})
}
