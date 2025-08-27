package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

func TestReaderFiltering(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logContent := `[2025-08-17T16:14:51.319Z] info  Info message
[2025-08-17T16:14:51.320Z] debug Debug message
[2025-08-17T16:14:51.321Z] warn  Warning message
[2025-08-17T16:14:51.322Z] error Error message
[21:26:07] [192.168.10.101] Extension Host Process exited
File not found: /usr/lib/workspace/lib/vscode/dist/index.js
Plain text error message`

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	assert.NilError(t, err)

	os.Setenv("WS_LOGGING_DIR", tempDir)
	os.Setenv("WS_LOGGING_MAIN_FILE", "test.log")
	defer func() {
		os.Unsetenv("WS_LOGGING_DIR")
		os.Unsetenv("WS_LOGGING_MAIN_FILE")
	}()

	tests := []struct {
		name        string
		levelFilter string
		expected    int
	}{
		{"no filter", "", 7},         // All lines including non-standard formats
		{"info filter", "info", 3},   // 1 matching structured line + 2 non-structured lines
		{"debug filter", "debug", 3}, // 1 matching structured line + 2 non-structured lines
		{"warn filter", "warn", 3},   // 1 matching structured line + 2 non-structured lines
		{"error filter", "error", 3}, // 1 matching structured line + 2 non-structured lines
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := NewReader(100, tt.levelFilter)
			assert.NilError(t, err)

			var buf bytes.Buffer
			err = reader.ReadLogs(&buf)
			assert.NilError(t, err)

			lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
			assert.Equal(t, tt.expected, len(lines))
		})
	}
}
