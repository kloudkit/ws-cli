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
