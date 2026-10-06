package info

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestFormatMemory(t *testing.T) {
	assert.Equal(t, formatMemory(1<<30, 0), "1.0 GiB (no limit)")
	assert.Equal(t, formatMemory(1<<30, 4<<30), "1.0 GiB / 4.0 GiB (25.0%)")
}
