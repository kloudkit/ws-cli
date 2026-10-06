package info

import (
	"bytes"
	"testing"

	"gotest.tools/v3/assert"
)

func TestShowVersionRendersResolvedVersionVerbatim(t *testing.T) {
	buf := &bytes.Buffer{}
	showVersionCmd.SetOut(buf)

	showVersionCmd.Run(showVersionCmd, []string{})

	assert.Equal(t, buf.String(), Version()+"\n")
}
