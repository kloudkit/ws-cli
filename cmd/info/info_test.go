package info

import (
	"bytes"
	"testing"

	"gotest.tools/v3/assert"
)

func TestVersionRendersResolvedVersionVerbatim(t *testing.T) {
	buf := &bytes.Buffer{}
	versionCmd.SetOut(buf)

	assert.NilError(t, versionCmd.RunE(versionCmd, nil))
	assert.Equal(t, buf.String(), Version()+"\n")
}

func TestUsage(t *testing.T) {
	assert.Equal(t, usage(1<<30, 4<<30), "1.0 GiB / 4.0 GiB (25.0%)")
}
