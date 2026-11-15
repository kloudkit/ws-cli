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
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

var Command = func(ctx context.Context, args []string) (*exec.Cmd, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, err
	}

	return exec.CommandContext(ctx, path, args...), nil
}

var (
	timeout     = 60 * time.Second
	waitDelay   = time.Second
	outputLimit = 64 << 10
)

var scrubbedEnv = []string{"NO_COLOR", "CLICOLOR_FORCE", "WS__INTERNAL_FORCE_COLOR"}

func Run(parent context.Context, args []string) (text string, failed bool) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	command, err := Command(ctx, args)
	if err != nil {
		return fmt.Sprintf("cannot run the command: %v", err), true
	}

	stdout, stderr := &tail{limit: outputLimit}, &tail{limit: outputLimit}

	command.Stdin = nil
	command.Stdout = stdout
	command.Stderr = stderr
	command.Env = scrubEnv(command.Env)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.WaitDelay = waitDelay
	command.Cancel = func() error { return killGroup(command) }

	err = command.Run()
	_ = killGroup(command)

	var (
		exit *exec.ExitError
		note string
	)

	switch {
	case err == nil, errors.Is(err, exec.ErrWaitDelay):
	case parent.Err() != nil:
		note, failed = "cancelled", true
	case ctx.Err() != nil:
		note, failed = fmt.Sprintf("timed out after %s", timeout), true
	case errors.As(err, &exit):
		failed = true
		if clean(string(stderr.bytes())) == "" {
			note = fmt.Sprintf("exit code %d", exit.ExitCode())
		}
	default:
		note, failed = fmt.Sprintf("cannot run the command: %v", err), true
	}

	if text = render(outputLimit, note, stdout, stderr); text == "" {
		text = "exit code 0"
	}

	return text, failed
}

func killGroup(command *exec.Cmd) error {
	if command.Process == nil {
		return nil
	}

	err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}

	return err
}

func scrubEnv(env []string) []string {
	if env == nil {
		env = os.Environ()
	}

	env = slices.DeleteFunc(slices.Clone(env), func(entry string) bool {
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
