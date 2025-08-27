package log

import (
	"bytes"
	"io"
	"testing"

	"github.com/kloudkit/ws-cli/internals/logger"
	"gotest.tools/v3/assert"
)

func TestWarnCommandInvokesLogWithFlags(t *testing.T) {
	var gotLevel, gotMsg string
	var gotIndent int
	var gotStamp bool
	called := 0

	original := logger.Log
	logger.Log = func(w io.Writer, level, message string, indent int, withStamp bool) {
		called++
		gotLevel = level
		gotMsg = message
		gotIndent = indent
		gotStamp = withStamp
	}
	defer func() { logger.Log = original }()

	buffer := new(bytes.Buffer)
	cmd := LogCmd
	cmd.SetOut(buffer)
	cmd.SetArgs([]string{"warn", "hello", "--indent", "2", "--stamp"})

	err := cmd.Execute()
	assert.NilError(t, err)
	assert.Equal(t, 1, called)
	assert.Equal(t, "warn", gotLevel)
	assert.Equal(t, "hello", gotMsg)
	assert.Equal(t, 2, gotIndent)
	assert.Assert(t, gotStamp)
}

func TestInfoCommandUsesPipeWhenFlagged(t *testing.T) {
	var gotLevel string
	var gotIndent int
	var gotStamp bool
	called := 0

	original := logger.Pipe
	logger.Pipe = func(r io.Reader, w io.Writer, level string, indent int, withStamp bool) {
		called++
		gotLevel = level
		gotIndent = indent
		gotStamp = withStamp
	}
	defer func() { logger.Pipe = original }()

	cmd := LogCmd
	cmd.SetIn(bytes.NewBufferString("foo\n"))
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetArgs([]string{"info", "--pipe", "--indent", "1", "--stamp"})

	err := cmd.Execute()
	assert.NilError(t, err)
	assert.Equal(t, 1, called)
	assert.Equal(t, "info", gotLevel)
	assert.Equal(t, 1, gotIndent)
	assert.Assert(t, gotStamp)
}

func TestStampCommandInvokesLog(t *testing.T) {
	called := 0
	var gotLevel, gotMsg string
	var gotIndent int
	var gotStamp bool

	original := logger.Log
	logger.Log = func(w io.Writer, level, message string, indent int, withStamp bool) {
		called++
		gotLevel = level
		gotMsg = message
		gotIndent = indent
		gotStamp = withStamp
	}
	defer func() { logger.Log = original }()

	cmd := LogCmd
	cmd.PersistentFlags().Set("pipe", "false")
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetArgs([]string{"stamp"})

	err := cmd.Execute()
	assert.NilError(t, err)
	assert.Equal(t, 1, called)
	assert.Equal(t, "", gotLevel)
	assert.Equal(t, "", gotMsg)
	assert.Equal(t, 0, gotIndent)
	assert.Assert(t, gotStamp)
}
