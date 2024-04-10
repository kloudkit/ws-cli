package log

import (
	"bytes"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestInfo(t *testing.T) {
	buffer := new(bytes.Buffer)

	log(buffer, "info", "This is my message", 0, false)

	assert.Equal(t, "info  This is my message\n", buffer.String())
}

func TestInfoWithStamp(t *testing.T) {
	buffer := new(bytes.Buffer)

	log(buffer, "info", "This has a stamp", 0, true)

	assert.Assert(
		t,
		cmp.Regexp(
			"^\\[\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}Z] info  This has a stamp\n$",
			buffer.String(),
		),
	)
}

func TestInfoWithIndent(t *testing.T) {
	buffer := new(bytes.Buffer)

	log(buffer, "info", "This is indented", 1, false)

	assert.Equal(t, "info    - This is indented\n", buffer.String())
}

func TestInfoWithStampAndIndent(t *testing.T) {
	buffer := new(bytes.Buffer)

	log(buffer, "info", "Stamped and indented", 2, true)

	assert.Assert(
		t,
		cmp.Regexp(
			"^\\[\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}Z] info      - Stamped and indented\n$",
			buffer.String(),
		),
	)
}

func TestPipe(t *testing.T) {
	buffer := new(bytes.Buffer)

	pipe(bytes.NewBufferString("foo\nbar\nbaz"), buffer, "debug", 2, true)

	assert.Assert(
		t,
		cmp.Regexp(
			`^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] debug     - foo\n`+
				`\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] debug     - bar\n`+
				`\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] debug     - baz\n$`,
			buffer.String(),
		),
	)
}

func TestStamp(t *testing.T) {
	buffer := new(bytes.Buffer)

	cmd := LogCmd

	cmd.SetOut(buffer)
	cmd.SetArgs([]string{"stamp"})
	cmd.Execute()

	assert.Assert(
		t,
		cmp.Regexp(
			"^\\[\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}Z] \n$",
			buffer.String(),
		),
	)
}

func TestCommand(t *testing.T) {
	buffer := new(bytes.Buffer)

	cmd := LogCmd

	cmd.SetOut(buffer)
	cmd.SetArgs([]string{"warn", "Stamped and indented", "--stamp", "--indent=2"})
	cmd.Execute()

	assert.Assert(
		t,
		cmp.Regexp(
			"^\\[\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}Z] warn      - Stamped and indented\n$",
			buffer.String(),
		),
	)
}
