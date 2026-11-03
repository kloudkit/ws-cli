package log

import (
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/cmd/clitest"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestLogCommand(t *testing.T) {
	const stamp = `^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] `

	cases := []struct {
		args, stdin, want string
	}{
		{"warn hello --indent 2 --stamp", "", stamp + `warn      - hello\n$`},
		{"info --pipe --indent 1 --stamp", "foo\n", stamp + `info    - foo\n$`},
		{"stamp --pipe=false", "", stamp + `\n$`},
	}

	for _, c := range cases {
		t.Run(c.args, func(t *testing.T) {
			out, err := clitest.Run(LogCmd, c.stdin, strings.Fields(c.args)...)
			assert.NilError(t, err)
			assert.Assert(t, cmp.Regexp(c.want, out))
		})
	}
}
