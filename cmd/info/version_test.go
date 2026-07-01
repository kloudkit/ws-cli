package info

import (
	"bytes"
	"testing"

	"gotest.tools/v3/assert"
)

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", "(devel)"},
		{"devel", "(devel)", "(devel)"},
		{"semver", "v1.2.3", "v1.2.3"},
		{"pseudo", "v0.0.0-20260101000000-abcdef123456", "v0.0.0-20260101000000-abcdef123456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, resolveVersion(tt.in), tt.want)
		})
	}
}

func TestShowVersionRendersResolvedVersionVerbatim(t *testing.T) {
	buf := &bytes.Buffer{}
	showVersionCmd.SetOut(buf)

	showVersionCmd.Run(showVersionCmd, []string{})

	assert.Equal(t, buf.String(), Version()+"\n")
}
