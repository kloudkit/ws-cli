package logger

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestLog(t *testing.T) {
	stamp := `\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] `

	tests := []struct {
		name      string
		message   string
		indent    int
		withStamp bool
		want      string
	}{
		{"Plain", "This is my message", 0, false, `info  This is my message`},
		{"WithStamp", "This has a stamp", 0, true, stamp + `info  This has a stamp`},
		{"WithIndent", "This is indented", 1, false, `info    - This is indented`},
		{"WithStampAndIndent", "Stamped and indented", 2, true, stamp + `info      - Stamped and indented`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := new(bytes.Buffer)

			Log(buffer, "info", tt.message, tt.indent, tt.withStamp)

			assert.Assert(t, cmp.Regexp("^"+tt.want+`\n$`, testutil.StripAnsi(buffer.String())))
		})
	}
}

func TestPipe(t *testing.T) {
	buffer := new(bytes.Buffer)

	Pipe(bytes.NewBufferString("foo\nbar\nbaz"), buffer, "debug", 2, true)

	assert.Assert(t, cmp.Regexp(`^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] debug     - foo\n`+
		`\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] debug     - bar\n`+
		`\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z] debug     - baz\n$`, testutil.StripAnsi(buffer.String())))
}

const fixtureLog = "[2026-10-11T10:00:00.000Z] \x1b[38;2;140;170;238minfo \x1b[m Starting\n" +
	"[2026-10-11T10:00:01.000Z] \x1b[38;2;115;121;148mdebug\x1b[m   - Loading\n" +
	"[2026-10-11T10:00:02.000Z] warn  Slow disk\n" +
	"[10:00:03] The editor is listening on port 8080\n" +
	"[2026-10-11T10:00:04.000Z] \x1b[1;38;2;231;130;132merror\x1b[m Failed one\n" +
	"[2026-10-11T10:00:05.000Z] info  Running\n" +
	"File not found: /usr/lib/workspace/index.js\n" +
	"[2026-10-11T10:00:06.000Z] error Failed two\n" +
	"[2026-10-11T10:00:07.000Z] \n" +
	"[2026-10-11T10:00:08.000Z] debug   - Done\n"

func _reader(t *testing.T, files map[string]string, target string, tail int, level string) (*Reader, string) {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		assert.NilError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}

	t.Setenv("WS_LOGGING_DIR", dir)
	t.Setenv("WS_LOGGING_MAIN_FILE", "workspace.log")
	t.Setenv("WS_LOGGING_METRICS_FILE", "metrics.log")
	t.Setenv("WS_LOGGING_DOCKER_FILE", "dockerd.log")

	reader, err := NewReader(tail, level, target)
	assert.NilError(t, err)

	return reader, filepath.Join(dir, "workspace.log")
}

func readFixture(t *testing.T, content string, tail int, level string) string {
	t.Helper()

	reader, _ := _reader(t, map[string]string{"workspace.log": content}, "main", tail, level)

	var buf bytes.Buffer
	assert.NilError(t, reader.ReadLogs(&buf, false))

	return buf.String()
}

func TestReaderFiltering(t *testing.T) {
	tests := []struct {
		name    string
		content string
		tail    int
		level   string
		want    []string
	}{
		{"AllLines", fixtureLog, 0, "", []string{
			"[2026-10-11T10:00:00.000Z] info  Starting",
			"[2026-10-11T10:00:01.000Z] debug   - Loading",
			"[2026-10-11T10:00:02.000Z] warn  Slow disk",
			"[10:00:03] The editor is listening on port 8080",
			"[2026-10-11T10:00:04.000Z] error Failed one",
			"[2026-10-11T10:00:05.000Z] info  Running",
			"File not found: /usr/lib/workspace/index.js",
			"[2026-10-11T10:00:06.000Z] error Failed two",
			"[2026-10-11T10:00:07.000Z] ",
			"[2026-10-11T10:00:08.000Z] debug   - Done",
		}},
		{"ErrorOnly", fixtureLog, 0, "error", []string{
			"[2026-10-11T10:00:04.000Z] error Failed one",
			"[2026-10-11T10:00:06.000Z] error Failed two",
		}},
		{"InfoOnly", fixtureLog, 0, "info", []string{
			"[2026-10-11T10:00:00.000Z] info  Starting",
			"[2026-10-11T10:00:05.000Z] info  Running",
		}},
		{"DebugOnly", fixtureLog, 0, "debug", []string{
			"[2026-10-11T10:00:01.000Z] debug   - Loading",
			"[2026-10-11T10:00:08.000Z] debug   - Done",
		}},
		{"TailOne", fixtureLog, 1, "", []string{
			"[2026-10-11T10:00:08.000Z] debug   - Done",
		}},
		{"TailThree", fixtureLog, 3, "", []string{
			"[2026-10-11T10:00:06.000Z] error Failed two",
			"[2026-10-11T10:00:07.000Z] ",
			"[2026-10-11T10:00:08.000Z] debug   - Done",
		}},
		{"TailCountsAfterLevel", fixtureLog, 1, "info", []string{
			"[2026-10-11T10:00:05.000Z] info  Running",
		}},
		{"TailBeyondMatches", fixtureLog, 5, "warn", []string{
			"[2026-10-11T10:00:02.000Z] warn  Slow disk",
		}},
		{"TailWholeFile", "a\nb\n", 5, "", []string{"a", "b"}},
		{"NoTrailingNewline", "a\nb\nc", 2, "", []string{"b", "c"}},
		{"NoTrailingNewlineAll", "a\nb\nc", 0, "", []string{"a", "b", "c"}},
		{"SingleLine", "only\n", 1, "", []string{"only"}},
		{"CRLF", "a\r\nb\r\n", 1, "", []string{"b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.StripAnsi(readFixture(t, tt.content, tt.tail, tt.level))
			assert.Equal(t, got, strings.Join(tt.want, "\n")+"\n")
		})
	}
}

func TestReaderTailSpansChunks(t *testing.T) {
	var content strings.Builder
	for i := range 5000 {
		fmt.Fprintf(&content, "[2026-10-11T10:00:00.000Z] info  line %d\n", i)
		if i%7 == 0 {
			fmt.Fprintf(&content, "[2026-10-11T10:00:00.000Z] error line %d\n", i)
		}
	}

	got := testutil.StripAnsi(readFixture(t, content.String(), 3, "error"))
	assert.Equal(t, got, "[2026-10-11T10:00:00.000Z] error line 4984\n"+
		"[2026-10-11T10:00:00.000Z] error line 4991\n"+
		"[2026-10-11T10:00:00.000Z] error line 4998\n")

	got = testutil.StripAnsi(readFixture(t, content.String(), 2, ""))
	assert.Equal(t, got, "[2026-10-11T10:00:00.000Z] error line 4998\n"+
		"[2026-10-11T10:00:00.000Z] info  line 4999\n")
}

func TestReaderRecolorsLevels(t *testing.T) {
	want := "[2026-10-11T10:00:02.000Z] " + styleMap["warn"].Render("warn") + "  Slow disk\n"
	assert.Equal(t, readFixture(t, fixtureLog, 1, "warn"), want)
}

func TestReaderFollowHonorsTail(t *testing.T) {
	reader, logPath := _reader(t, map[string]string{"workspace.log": "a\nb\nc\n"}, "main", 2, "")

	out := &syncBuffer{}
	go reader.ReadLogs(out, true)

	waitFor(t, func() bool { return out.String() == "b\nc\n" })

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o644)
	assert.NilError(t, err)
	_, err = file.WriteString("d\n")
	assert.NilError(t, err)
	assert.NilError(t, file.Close())

	waitFor(t, func() bool { return out.String() == "b\nc\nd\n" })
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	assert.Assert(t, cond(), "condition not met before deadline")
}

func TestStrip(t *testing.T) {
	input := "\x1b[1;31mred\x1b[0m text\n" +
		"\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\ here\n" +
		strings.Repeat("x", 100_000) + "\n" +
		"truncated \x1b[38;2;1"

	var out bytes.Buffer
	assert.NilError(t, Strip(strings.NewReader(input), &out))

	assert.Equal(t, out.String(), "red text\nlink here\n"+strings.Repeat("x", 100_000)+"\ntruncated ")
}

func TestReaderTargets(t *testing.T) {
	files := map[string]string{
		"workspace.log": "main marker line\n",
		"metrics.log":   "metrics marker line\n",
		"dockerd.log":   "",
	}

	for target, want := range map[string]string{"main": "main marker line", "": "main marker line", "metrics": "metrics marker line", "docker": ""} {
		reader, _ := _reader(t, files, target, 0, "")

		var buf bytes.Buffer
		assert.NilError(t, reader.ReadLogs(&buf, false))
		assert.Equal(t, want, strings.TrimSpace(buf.String()), target)
	}

	t.Setenv("WS_LOGGING_METRICS_FILE", "absent.log")
	_, err := NewReader(0, "", "metrics")
	assert.ErrorContains(t, err, "log file not found")
}
