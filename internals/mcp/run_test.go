package mcp

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/kloudkit/ws-cli/internals/proc"
	"gotest.tools/v3/assert"
)

func override[T any](t *testing.T, target *T, value T) {
	t.Helper()

	previous := *target
	*target = value

	t.Cleanup(func() { *target = previous })
}

func TestRun(t *testing.T) {
	override(t, &proc.Executable, func() (string, error) { return "/bin/sh", nil })

	run := func(script string) (string, bool) { return Run(context.Background(), []string{"-c", script}) }

	cases := map[string]struct {
		script string
		want   string
		failed bool
	}{
		"Output":                   {`echo out; echo err >&2`, "out\nerr", false},
		"SilentSuccess":            {`true`, "exit code 0", false},
		"NonZeroExit":              {`echo partial; printf '\033[1;31mboom\033[0m\n' >&2; exit 3`, "partial\nboom", true},
		"SilentExitYieldsTheCode":  {`exit 1`, "exit code 1", true},
		"BlankStderrYieldsTheCode": {`echo "  " >&2; exit 1`, "exit code 1", true},
		"EmptyStderrYieldsTheCode": {`echo only stdout; exit 2`, "only stdout\nexit code 2", true},
		"StripsAnsi":               {`printf '\033[38;2;140;170;238mblue\033[m plain \033]8;;https://example.com\033\\link\033]8;;\033\\   \n\ncut \033[38;2'`, "blue plain link\n\ncut", false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			text, failed := run(tc.script)

			assert.Equal(t, text, tc.want)
			assert.Equal(t, failed, tc.failed)
		})
	}

	t.Run("ScrubsColorEnvAndKeepsTheRest", func(t *testing.T) {
		t.Setenv("CLICOLOR_FORCE", "1")
		t.Setenv("WS__INTERNAL_FORCE_COLOR", "1")
		t.Setenv("NO_COLOR", "")
		t.Setenv("KEPT", "from the parent")

		text, _ := run(`echo "$NO_COLOR|${CLICOLOR_FORCE-unset}|${WS__INTERNAL_FORCE_COLOR-unset}|$KEPT"`)

		assert.Equal(t, text, "1|unset|unset|from the parent")
	})

	t.Run("InvalidUTF8", func(t *testing.T) {
		text, _ := run(`printf 'a\377\376b\n'`)

		assert.Assert(t, utf8.ValidString(text))
		assert.Assert(t, strings.HasPrefix(text, "a") && strings.HasSuffix(text, "b"), "%q", text)
	})

	const flood = `printf x; yes é | head -n 2000 | tr -d '\n'`

	t.Run("TruncatesKeepingTheTail", func(t *testing.T) {
		override(t, &outputLimit, 1023)

		text, failed := run(flood)

		assert.Equal(t, text, "[output truncated: showing the last 1022 of 4001 bytes]\n"+strings.Repeat("é", 511))
		assert.Assert(t, !failed)
	})

	t.Run("UnderTheCapIsWhole", func(t *testing.T) {
		override(t, &outputLimit, 4001)

		text, _ := run(flood)

		assert.Equal(t, text, "x"+strings.Repeat("é", 2000))
	})

	t.Run("TimedOut", func(t *testing.T) {
		override(t, &timeout, 300*time.Millisecond)

		text, failed := run(`echo started; sleep 60`)

		assert.Equal(t, text, "started\ntimed out after 300ms")
		assert.Assert(t, failed)
	})

	t.Run("Cancelled", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		text, failed := Run(ctx, []string{"-c", `echo started; sleep 60`})

		assert.Equal(t, text, "started\ncancelled")
		assert.Assert(t, failed)
	})

	t.Run("CannotStart", func(t *testing.T) {
		override(t, &proc.Executable, func() (string, error) { return "/nonexistent/ws-cli", nil })

		text, failed := run(`true`)

		assert.Assert(t, strings.HasPrefix(text, "cannot run the command:"), text)
		assert.Assert(t, failed)
	})
}
