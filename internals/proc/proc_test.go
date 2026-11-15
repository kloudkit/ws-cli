package proc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func override[T any](t *testing.T, target *T, value T) {
	t.Helper()

	previous := *target
	*target = value

	t.Cleanup(func() { *target = previous })
}

func sh(timeout time.Duration, script string, args ...string) (string, error) {
	out, err := Output(timeout, "/bin/sh", append([]string{"-c", script, "sh"}, args...)...)

	return string(out), err
}

func assertDies(t *testing.T, out string) {
	t.Helper()

	pid, err := strconv.Atoi(strings.TrimSpace(out))
	assert.NilError(t, err, out)

	for range 200 {
		stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if _, state, _ := strings.Cut(string(stat), ") "); err != nil || strings.HasPrefix(state, "Z") {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("process %d is still running", pid)
}

func TestRun(t *testing.T) {
	override(t, &waitDelay, 200*time.Millisecond)

	t.Run("PassesArgsVerbatimWithEmptyStdin", func(t *testing.T) {
		out, err := sh(time.Minute, `printf '%s|' "$@"; wc -c`, "show", "--", "--value", "a b", "")

		assert.NilError(t, err)
		assert.Equal(t, out, "show|--|--value|a b||0\n")
	})

	t.Run("ReportsTheExitStatus", func(t *testing.T) {
		var exit *exec.ExitError

		_, err := sh(time.Minute, "exit 3")
		assert.Assert(t, errors.As(err, &exit) && exit.ExitCode() == 3, "%v", err)
	})

	t.Run("CannotStart", func(t *testing.T) {
		var exit *exec.ExitError

		_, err := Output(time.Minute, "/nonexistent/ws-cli")
		assert.Assert(t, err != nil && !errors.As(err, &exit), "%v", err)
	})

	t.Run("TimesOutAChild", func(t *testing.T) {
		start := time.Now()
		_, err := Output(300*time.Millisecond, "/bin/sleep", "60")

		assert.ErrorIs(t, err, ErrTimeout)
		assert.Assert(t, time.Since(start) < 3*time.Second, "took %s", time.Since(start))
	})

	t.Run("TimesOutAGrandchild", func(t *testing.T) {
		start := time.Now()
		out, err := sh(500*time.Millisecond, "sleep 60 & echo $!; wait")

		assert.ErrorIs(t, err, ErrTimeout)
		assert.Assert(t, time.Since(start) < 3*time.Second, "took %s", time.Since(start))
		assertDies(t, out)
	})

	t.Run("OrphanHoldingThePipe", func(t *testing.T) {
		start := time.Now()
		out, err := sh(20*time.Second, "sleep 60 & echo $!")

		assert.NilError(t, err)
		assert.Assert(t, time.Since(start) < 10*time.Second, "took %s", time.Since(start))
		assertDies(t, out)
	})

	t.Run("FailingChildLeavesNoOrphan", func(t *testing.T) {
		var exit *exec.ExitError

		out, err := sh(20*time.Second, "sleep 60 & echo $!; exit 5")
		assert.Assert(t, errors.As(err, &exit) && exit.ExitCode() == 5, "%v", err)
		assertDies(t, out)
	})

	t.Run("SuccessAtTheDeadlineIsSuccess", func(t *testing.T) {
		override(t, &waitDelay, 20*time.Second)
		override(t, &kill, func(int, syscall.Signal) error { return syscall.ESRCH })

		out, err := sh(50*time.Millisecond, "sleep 0.3; echo done")

		assert.NilError(t, err)
		assert.Equal(t, out, "done\n")
	})

	t.Run("CallerDeadlineIsNotOurTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		err := Cmd{Path: "/bin/sleep", Args: []string{"60"}, Timeout: time.Minute}.Run(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}
