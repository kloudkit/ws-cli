package env

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func TestString(t *testing.T) {
	os.Unsetenv("FOO")

	assert.Equal(t, "bar", String("FOO", "bar"))

	os.Setenv("FOO", "baz")
	defer os.Unsetenv("FOO")

	assert.Equal(t, "baz", String("FOO", "bar"))
}

func TestMustString(t *testing.T) {
	os.Unsetenv("FOO")

	assert.Assert(t, func() (result bool) {
		defer func() {
			result = recover() != nil
		}()

		MustString("FOO")

		return false
	}(), "expected panic when env var missing and no fallback")
}

func TestMustStringFallback(t *testing.T) {
	os.Unsetenv("FOO")

	assert.Equal(t, "qux", MustString("FOO", "qux"))
}

func TestGetAll(t *testing.T) {
	os.Setenv("TEST_KEY1", "value1")
	os.Setenv("TEST_KEY2", "value2")

	defer func() {
		os.Unsetenv("TEST_KEY1")
		os.Unsetenv("TEST_KEY2")
	}()

	envMap := GetAll()

	assert.Equal(t, "value1", envMap["TEST_KEY1"])
	assert.Equal(t, "value2", envMap["TEST_KEY2"])
	assert.Assert(t, len(envMap) > 0, "expected GetAll() to return non-empty map")
}
