package mcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/kloudkit/ws-cli/internals/proc"
)

var (
	timeout     = 60 * time.Second
	outputLimit = 64 << 10
)

var scrubbedEnv = []string{"NO_COLOR", "CLICOLOR_FORCE", "WS__INTERNAL_FORCE_COLOR"}

func Run(ctx context.Context, args []string) (text string, failed bool) {
	path, err := proc.Executable()
	if err != nil {
		return fmt.Sprintf("cannot run the command: %v", err), true
	}

	stdout, stderr := &tail{limit: outputLimit}, &tail{limit: outputLimit}

	err = proc.Cmd{Path: path, Args: args, Env: scrubEnv(), Stdout: stdout, Stderr: stderr, Timeout: timeout}.Run(ctx)

	var (
		exit *exec.ExitError
		note string
	)

	switch {
	case err == nil:
	case errors.Is(err, proc.ErrTimeout):
		note = fmt.Sprintf("timed out after %s", timeout)
	case ctx.Err() != nil:
		note = "cancelled"
	case !errors.As(err, &exit):
		note = fmt.Sprintf("cannot run the command: %v", err)
	case clean(string(stderr.bytes())) == "":
		note = fmt.Sprintf("exit code %d", exit.ExitCode())
	}

	if text = render(outputLimit, note, stdout, stderr); text == "" {
		text = "exit code 0"
	}

	return text, err != nil
}

func scrubEnv() []string {
	env := slices.DeleteFunc(os.Environ(), func(entry string) bool {
		key, _, _ := strings.Cut(entry, "=")
		return slices.Contains(scrubbedEnv, key)
	})

	return append(env, "NO_COLOR=1")
}

type tail struct {
	mu    sync.Mutex
	limit int
	data  []byte
	total int
}

func (t *tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.total += len(p)
	t.data = append(t.data, p...)

	if len(t.data) > 2*t.limit {
		t.data = append(t.data[:0], t.data[len(t.data)-t.limit:]...)
	}

	return len(p), nil
}

func (t *tail) size() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.total
}

func (t *tail) bytes() []byte {
	t.mu.Lock()
	defer t.mu.Unlock()

	return slices.Clone(t.data)
}

func render(limit int, note string, streams ...*tail) string {
	var (
		data  []byte
		total int
	)

	for _, stream := range streams {
		data = append(data, stream.bytes()...)
		total += stream.size()
	}

	if len(data) > limit {
		data = data[len(data)-limit:]
	}

	var parts []string

	if total > len(data) {
		for skipped := 0; skipped < utf8.UTFMax-1 && len(data) > 0 && !utf8.RuneStart(data[0]); skipped++ {
			data = data[1:]
		}

		parts = append(parts, fmt.Sprintf("[output truncated: showing the last %d of %d bytes]", len(data), total))
	}

	if body := clean(string(data)); body != "" {
		parts = append(parts, body)
	}

	if note != "" {
		parts = append(parts, note)
	}

	return strings.Join(parts, "\n")
}

func clean(text string) string {
	lines := strings.Split(strings.ToValidUTF8(strings.ReplaceAll(ansi.Strip(text), "\x1b", ""), "\uFFFD"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}

	return strings.Trim(strings.Join(lines, "\n"), "\n")
}
