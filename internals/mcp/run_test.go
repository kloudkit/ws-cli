package mcp

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"gotest.tools/v3/assert"
)

const helperEnv = "MCP_TEST_HELPER"

func TestHelperProcess(t *testing.T) {
	script := os.Getenv(helperEnv)
	if script == "" {
		return
	}

	args := strings.Split(script, "\x1f")

	switch args[0] {
	case "env":
		for _, key := range []string{"NO_COLOR", "CLICOLOR_FORCE", "WS__INTERNAL_FORCE_COLOR", "KEPT"} {
			value, set := os.LookupEnv(key)
			fmt.Printf("%s=%s,%t\n", key, value, set)
		}
	case "stdin":
		data, err := io.ReadAll(os.Stdin)
		fmt.Printf("stdin=%d,%v\n", len(data), err)
	case "fail":
		fmt.Println("partial")
		fmt.Fprintln(os.Stderr, "\x1b[1;31mboom\x1b[0m")
		os.Exit(3)
	case "silent":
		os.Exit(1)
	case "blank-stderr":
		fmt.Fprintln(os.Stderr, "  ")
		os.Exit(1)
	case "quiet":
	case "stdout-fail":
		fmt.Println("only stdout")
		os.Exit(2)
	case "ansi":
		fmt.Print("\x1b[38;2;140;170;238mblue\x1b[m plain \x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\   \n\ncut \x1b[38;2")
	case "flood":
		fmt.Print("x" + strings.Repeat("é", 2000))
	case "binary":
		os.Stdout.Write([]byte{'a', 0xff, 0xfe, 'b', '\n'})
	case "sleep":
		fmt.Println("started")
		time.Sleep(time.Minute)
	case "grandchild":
		child := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
		child.Env = append(os.Environ(), helperEnv+"=sleep")
		child.Stdout, child.Stderr = os.Stdout, os.Stderr

		if err := child.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(4)
		}

		fmt.Printf("pid=%d\n", child.Process.Pid)

		switch args[1] {
		case "wait":
			_ = child.Wait()
		case "fail":
			os.Exit(5)
		}
	default:
		fmt.Println(strings.Join(args, "|"))
	}

	os.Exit(0)
}

func useHelper(t *testing.T) {
	t.Helper()

	previous := Command
	Command = func(ctx context.Context, args []string) (*exec.Cmd, error) {
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHelperProcess$")
		command.Env = append(os.Environ(), helperEnv+"="+strings.Join(args, "\x1f"))

		return command, nil
	}

	t.Cleanup(func() { Command = previous })
}

func override[T any](t *testing.T, target *T, value T) {
	t.Helper()

	previous := *target
	*target = value

	t.Cleanup(func() { *target = previous })
}

func alive(pid int) bool {
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return false
	}

	_, state, _ := strings.Cut(string(stat), ") ")

	return !strings.HasPrefix(state, "Z")
}

func grandchildPID(t *testing.T, text string) int {
	t.Helper()

	_, rest, found := strings.Cut(text, "pid=")
	assert.Assert(t, found, text)

	pid, err := strconv.Atoi(strings.Fields(rest)[0])
	assert.NilError(t, err)

	return pid
}

func assertDies(t *testing.T, pid int) {
	t.Helper()

	for range 200 {
		if !alive(pid) {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("process %d is still running", pid)
}

func TestRun(t *testing.T) {
	useHelper(t)

	run := func(args ...string) (string, bool) { return Run(context.Background(), args) }

	t.Run("PassesArgsVerbatim", func(t *testing.T) {
		text, failed := run("show", "env", "--", "--value", "a b", "")

		assert.Equal(t, text, "show|env|--|--value|a b|")
		assert.Assert(t, !failed)
	})

	t.Run("ScrubsColorEnv", func(t *testing.T) {
		t.Setenv("CLICOLOR_FORCE", "1")
		t.Setenv("WS__INTERNAL_FORCE_COLOR", "1")
		t.Setenv("NO_COLOR", "")
		t.Setenv("KEPT", "yes")

		text, failed := run("env")

		assert.Equal(t, text, "NO_COLOR=1,true\nCLICOLOR_FORCE=,false\nWS__INTERNAL_FORCE_COLOR=,false\nKEPT=yes,true")
		assert.Assert(t, !failed)
	})

	t.Run("EmptyStdin", func(t *testing.T) {
		text, failed := run("stdin")

		assert.Equal(t, text, "stdin=0,<nil>")
		assert.Assert(t, !failed)
	})

	t.Run("NonZeroExit", func(t *testing.T) {
		text, failed := run("fail")

		assert.Equal(t, text, "partial\nboom")
		assert.Assert(t, failed)
	})

	t.Run("SilentExitYieldsTheCode", func(t *testing.T) {
		text, failed := run("silent")

		assert.Equal(t, text, "exit code 1")
		assert.Assert(t, failed)
	})

	t.Run("BlankStderrYieldsTheCode", func(t *testing.T) {
		text, failed := run("blank-stderr")

		assert.Equal(t, text, "exit code 1")
		assert.Assert(t, failed)
	})

	t.Run("EmptyStderrYieldsTheCode", func(t *testing.T) {
		text, failed := run("stdout-fail")

		assert.Equal(t, text, "only stdout\nexit code 2")
		assert.Assert(t, failed)
	})

	t.Run("SilentSuccess", func(t *testing.T) {
		text, failed := run("quiet")

		assert.Equal(t, text, "exit code 0")
		assert.Assert(t, !failed)
	})

	t.Run("StripsAnsi", func(t *testing.T) {
		t.Setenv("CLICOLOR_FORCE", "1")

		text, failed := run("ansi")

		assert.Equal(t, text, "blue plain link\n\ncut")
		assert.Assert(t, !failed)
	})

	t.Run("InvalidUTF8", func(t *testing.T) {
		text, _ := run("binary")

		assert.Assert(t, utf8.ValidString(text))
		assert.Assert(t, strings.HasPrefix(text, "a") && strings.HasSuffix(text, "b"), "%q", text)
	})

	t.Run("TruncatesKeepingTheTail", func(t *testing.T) {
		override(t, &outputLimit, 1023)

		text, failed := run("flood")

		assert.Equal(t, text, "[output truncated: showing the last 1022 of 4001 bytes]\n"+strings.Repeat("é", 511))
		assert.Assert(t, utf8.ValidString(text))
		assert.Assert(t, !failed)
	})

	t.Run("UnderTheCapIsWhole", func(t *testing.T) {
		override(t, &outputLimit, 4001)

		text, _ := run("flood")

		assert.Equal(t, text, "x"+strings.Repeat("é", 2000))
	})

	t.Run("TimesOutAChild", func(t *testing.T) {
		override(t, &timeout, 300*time.Millisecond)
		override(t, &waitDelay, 200*time.Millisecond)

		start := time.Now()
		text, failed := run("sleep")

		assert.Assert(t, time.Since(start) < timeout+waitDelay+2*time.Second, "took %s", time.Since(start))
		assert.Equal(t, text, "started\ntimed out after 300ms")
		assert.Assert(t, failed)
	})

	t.Run("TimesOutAGrandchild", func(t *testing.T) {
		override(t, &timeout, 500*time.Millisecond)
		override(t, &waitDelay, 200*time.Millisecond)

		start := time.Now()
		text, failed := run("grandchild", "wait")

		assert.Assert(t, time.Since(start) < timeout+waitDelay+2*time.Second, "took %s", time.Since(start))
		assert.Assert(t, strings.HasSuffix(text, "timed out after 500ms"), text)
		assert.Assert(t, failed)
		assertDies(t, grandchildPID(t, text))
	})

	t.Run("OrphanHoldingThePipe", func(t *testing.T) {
		override(t, &timeout, 20*time.Second)
		override(t, &waitDelay, 200*time.Millisecond)

		start := time.Now()
		text, failed := run("grandchild", "exit")

		assert.Assert(t, time.Since(start) < 10*time.Second, "took %s", time.Since(start))
		assert.Assert(t, !failed, text)
		assertDies(t, grandchildPID(t, text))
	})

	t.Run("FailingChildLeavesNoOrphan", func(t *testing.T) {
		override(t, &timeout, 20*time.Second)
		override(t, &waitDelay, 200*time.Millisecond)

		start := time.Now()
		text, failed := run("grandchild", "fail")

		assert.Assert(t, time.Since(start) < 10*time.Second, "took %s", time.Since(start))
		assert.Assert(t, failed, text)
		assert.Assert(t, strings.HasSuffix(text, "exit code 5"), text)
		assertDies(t, grandchildPID(t, text))
	})

	t.Run("SuccessAtTheDeadlineIsSuccess", func(t *testing.T) {
		helper := Command
		override(t, &Command, func(ctx context.Context, args []string) (*exec.Cmd, error) {
			return helper(context.WithoutCancel(ctx), args)
		})
		override(t, &timeout, time.Millisecond)

		text, failed := run("show")

		assert.Equal(t, text, "show")
		assert.Assert(t, !failed)
	})

	t.Run("CallerDeadlineIsNotOurTimeout", func(t *testing.T) {
		override(t, &waitDelay, 200*time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		text, failed := Run(ctx, []string{"sleep"})

		assert.Equal(t, text, "started\ncancelled")
		assert.Assert(t, failed)
	})

	t.Run("InheritsTheParentEnv", func(t *testing.T) {
		override(t, &Command, func(ctx context.Context, args []string) (*exec.Cmd, error) {
			return exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHelperProcess$"), nil
		})
		t.Setenv(helperEnv, "env")
		t.Setenv("KEPT", "from the parent")
		t.Setenv("CLICOLOR_FORCE", "1")

		text, failed := run("ignored")

		assert.Equal(t, text, "NO_COLOR=1,true\nCLICOLOR_FORCE=,false\nWS__INTERNAL_FORCE_COLOR=,false\nKEPT=from the parent,true")
		assert.Assert(t, !failed)
	})

	t.Run("Cancelled", func(t *testing.T) {
		override(t, &waitDelay, 200*time.Millisecond)

		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(300*time.Millisecond, cancel)

		start := time.Now()
		text, failed := Run(ctx, []string{"sleep"})

		assert.Assert(t, time.Since(start) < 10*time.Second, "took %s", time.Since(start))
		assert.Assert(t, strings.HasSuffix(text, "cancelled"), text)
		assert.Assert(t, failed)
	})

	t.Run("CannotStart", func(t *testing.T) {
		override(t, &Command, func(ctx context.Context, args []string) (*exec.Cmd, error) {
			return exec.CommandContext(ctx, "/nonexistent/ws-cli", args...), nil
		})

		text, failed := run("info", "version")

		assert.Assert(t, strings.HasPrefix(text, "cannot run the command:"), text)
		assert.Assert(t, failed)
	})
}

func TestCommandRunsThisExecutable(t *testing.T) {
	command, err := Command(context.Background(), []string{"info", "version"})
	assert.NilError(t, err)

	self, err := os.Executable()
	assert.NilError(t, err)
	assert.Equal(t, command.Path, self)
	assert.DeepEqual(t, command.Args[1:], []string{"info", "version"})
}
