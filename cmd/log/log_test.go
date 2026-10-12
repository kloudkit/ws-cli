package log

import (
	"bytes"
	"testing"

	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestLogCommand(t *testing.T) {
	t.Run("WarnInvokesLogWithFlags", func(t *testing.T) {
		buffer := new(bytes.Buffer)
		cmd := LogCmd
		cmd.SetOut(buffer)
		cmd.SetArgs([]string{"warn", "hello", "--indent", "2", "--stamp"})

		err := cmd.Execute()
		assert.NilError(t, err)
		assert.Assert(t, cmp.Regexp(
			`^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] warn      - hello\n$`,
			testutil.StripAnsi(buffer.String()),
		))
	})

	t.Run("InfoUsesPipeWhenFlagged", func(t *testing.T) {
		buffer := new(bytes.Buffer)
		cmd := LogCmd
		cmd.SetIn(bytes.NewBufferString("foo\n"))
		cmd.SetOut(buffer)
		cmd.SetArgs([]string{"info", "--pipe", "--indent", "1", "--stamp"})

		err := cmd.Execute()
		assert.NilError(t, err)
		assert.Assert(t, cmp.Regexp(
			`^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] info    - foo\n$`,
			testutil.StripAnsi(buffer.String()),
		))
	})

	t.Run("StampInvokesLog", func(t *testing.T) {
		buffer := new(bytes.Buffer)
		cmd := LogCmd
		cmd.PersistentFlags().Set("pipe", "false")
		cmd.SetOut(buffer)
		cmd.SetArgs([]string{"stamp"})

		err := cmd.Execute()
		assert.NilError(t, err)
		assert.Assert(t, cmp.Regexp(
			`^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] \n$`,
			testutil.StripAnsi(buffer.String()),
		))
	})
}
