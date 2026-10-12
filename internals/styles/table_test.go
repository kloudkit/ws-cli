package styles

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"gotest.tools/v3/assert"
)

func TestTableFitsWidth(t *testing.T) {
	tbl := NewTable("Path", "Language", "Active", "Dirty").Rows(
		[]string{"/workspace/kloudkit/ws-cli/go.work", "go.work", "", ""},
		[]string{"/workspace/kloudkit/ws-cli/internals/styles/table.go", "go", "✓", "✓"},
		[]string{"/workspace/kloudkit/ws-bmad/_bmad-output/implementation-artifacts/spec-6-14-ws-cli-output-consistency.md", "markdown", "", ""},
	)

	got := ansi.Strip(tbl.RenderWidth(80))

	want := strings.Join([]string{
		"  ╭────────────────────────────────────────────────┬──────────┬────────┬───────╮",
		"  │ Path                                           │ Language │ Active │ Dirty │",
		"  ├────────────────────────────────────────────────┼──────────┼────────┼───────┤",
		"  │ /workspace/kloudkit/ws-cli/go.work             │ go.work  │        │       │",
		"  │ /workspace/kloudkit/ws-cli/internals/styles/   │ go       │ ✓      │ ✓     │",
		"  │ table.go                                       │          │        │       │",
		"  │ /workspace/kloudkit/ws-bmad/_bmad-output/      │ markdown │        │       │",
		"  │ implementation-artifacts/                      │          │        │       │",
		"  │ spec-6-14-ws-cli-output-consistency.md         │          │        │       │",
		"  ╰────────────────────────────────────────────────┴──────────┴────────┴───────╯",
	}, "\n")

	assert.Equal(t, got, want)
}

func TestTableWrapsWidestColumnOnWords(t *testing.T) {
	tbl := NewTable().Rows(
		[]string{"WS_TERMINAL_PROMPT_HIDE_DOCKER_CONTEXT", "1"},
		[]string{"WS_ZSH_PLUGINS", "1password artisan auto-venv docker encode64 fzf-history git golang helm kubectl npm pip python ssh-agent yarn zsh-autosuggestions"},
	)

	got := ansi.Strip(tbl.RenderWidth(80))

	for line := range strings.SplitSeq(got, "\n") {
		assert.Assert(t, ansi.StringWidth(line) <= 80, "line too wide: %q", line)
	}

	assert.Assert(t, strings.Contains(got, "│ WS_TERMINAL_PROMPT_HIDE_DOCKER_CONTEXT │ 1 "))
	assert.Assert(t, strings.Contains(got, "zsh-autosuggestions"))
}

func TestTableSizedToContentWithoutLimit(t *testing.T) {
	got := ansi.Strip(NewTable().Rows([]string{"Status", "Active"}).RenderWidth(0))

	assert.Equal(t, got, "  ╭────────┬────────╮\n  │ Status │ Active │\n  ╰────────┴────────╯")
}

func TestTableNarrowTerminal(t *testing.T) {
	tbl := NewTable("Severity", "Location", "Source", "Message").Rows(
		[]string{"Warning", "14:27", "unusedparams", "unusedparams: unused parameter: col and a much longer explanation"},
	)

	for _, width := range []int{80, 60, 40} {
		for line := range strings.SplitSeq(ansi.Strip(tbl.RenderWidth(width)), "\n") {
			assert.Assert(t, ansi.StringWidth(line) <= width, "width %d: %q", width, line)
		}
	}
}
