package log

import (
	"bytes"
	"regexp"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func stripAnsi(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)

	return re.ReplaceAllString(s, "")
}

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
			stripAnsi(buffer.String()),
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
			stripAnsi(buffer.String()),
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
			stripAnsi(buffer.String()),
		))
	})
}
