package log

import (
	"bytes"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestLog(t *testing.T) {
	buffer := new(bytes.Buffer)

	Log(buffer, "info", "This is my message", 0, false)

	assert.Equal(t, "info  This is my message\n", buffer.String())
}

func TestLogWithStamp(t *testing.T) {
	buffer := new(bytes.Buffer)

	Log(buffer, "info", "This has a stamp", 0, true)

	assert.Assert(t, cmp.Regexp(`^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] info  This has a stamp\n$`, buffer.String()))
}

func TestLogWithIndent(t *testing.T) {
	buffer := new(bytes.Buffer)

	Log(buffer, "info", "This is indented", 1, false)

	assert.Equal(t, "info    - This is indented\n", buffer.String())
}

func TestLogWithStampAndIndent(t *testing.T) {
	buffer := new(bytes.Buffer)

	Log(buffer, "info", "Stamped and indented", 2, true)

	assert.Assert(t, cmp.Regexp(`^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] info      - Stamped and indented\n$`, buffer.String()))
}

func TestPipe(t *testing.T) {
	buffer := new(bytes.Buffer)

	Pipe(bytes.NewBufferString("foo\nbar\nbaz"), buffer, "debug", 2, true)

	assert.Assert(t, cmp.Regexp(`^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] debug     - foo\n`+
		`\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] debug     - bar\n`+
		`\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] debug     - baz\n$`, buffer.String()))
}
