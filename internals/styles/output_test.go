package styles

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"gotest.tools/v3/assert"
)

func plain(fn func(*bytes.Buffer)) string {
	var buf bytes.Buffer
	fn(&buf)

	return ansi.Strip(buf.String())
}

func TestPrinters(t *testing.T) {
	cases := map[string]struct {
		fn   func(*bytes.Buffer)
		want string
	}{
		"Title":      {func(b *bytes.Buffer) { PrintTitle(b, "Template ruff") }, "\n  Template ruff\n\n"},
		"TitleCount": {func(b *bytes.Buffer) { PrintTitle(b, "gh", 3) }, "\n  gh (3)\n\n"},
		"TitleNote":  {func(b *bytes.Buffer) { PrintTitle(b, "server.port", "WS_SERVER_PORT") }, "\n  server.port WS_SERVER_PORT\n\n"},
		"KeyValues": {
			func(b *bytes.Buffer) {
				PrintKeyValues(b, Pair{"URL", "http://localhost:8080"}, Pair{"Root", "/workspace"}, Pair{"Value", "line one\nline two"})
			},
			"  URL    http://localhost:8080\n  Root   /workspace\n  Value  line one\n         line two\n",
		},
		"List": {
			func(b *bytes.Buffer) {
				PrintList(b, []DescriptionItem{{"gh", "Install gh CLI"}, {"terraform", "Install Terraform"}})
			},
			"  • gh        — Install gh CLI\n  • terraform — Install Terraform\n",
		},
		"Bullets": {func(b *bytes.Buffer) { PrintBullets(b, []string{"a", "b"}) }, "  • a\n  • b\n"},
		"Hints": {
			func(b *bytes.Buffer) {
				PrintHints(b, Hint{"ws feature install <name>", "Install a feature"}, Hint{"ws feature info <name>", "View feature details"})
			},
			"\n  Quick actions\n    ws feature install <name>  Install a feature\n    ws feature info <name>     View feature details\n",
		},
		"Empty":          {func(b *bytes.Buffer) { PrintEmpty(b, "No open editors") }, "\n  ○ No open editors\n"},
		"Success":        {func(b *bytes.Buffer) { PrintSuccess(b, "Done") }, "  ✓ Done\n"},
		"SuccessDetails": {func(b *bytes.Buffer) { PrintSuccess(b, "Done", Pair{"Output", "/tmp/k"}) }, "\n  ✓ Done\n\n  Output  /tmp/k\n"},
		"Warning":        {func(b *bytes.Buffer) { PrintWarning(b, "Careful") }, "  ⚠ Careful\n"},
		"Text":           {func(b *bytes.Buffer) { PrintText(b, "a\n\nb") }, "  a\n\n  b\n"},
		"Code":           {func(b *bytes.Buffer) { PrintCode(b, "a: 1  \n\nb: 2\n") }, "  │ a: 1\n  │\n  │ b: 2\n"},
		"MarkdownEmpty":  {func(b *bytes.Buffer) { assert.NilError(t, RenderMarkdown(b, "   \n\n")) }, ""},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) { assert.Equal(t, plain(tc.fn), tc.want) })
	}
}

func TestRenderMarkdown(t *testing.T) {
	got := plain(func(b *bytes.Buffer) { assert.NilError(t, RenderMarkdown(b, "Accepts a **space-delimited** list.")) })

	assert.Assert(t, strings.Contains(got, "space-delimited"))
	assert.Assert(t, !strings.Contains(got, " \n"))
}

func TestRenderMarkdownKeepsTrailingReset(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1")

	var buf bytes.Buffer
	assert.NilError(t, RenderMarkdown(NewWriter(&buf), "Port on which the web server listens."))

	got := strings.TrimRight(buf.String(), "\n")
	assert.Assert(t, strings.HasSuffix(got, "m") && strings.Contains(got[strings.LastIndex(got, "listens."):], "\x1b["), "%q", got)
	assert.Assert(t, !strings.Contains(ansi.Strip(got), " \n"))
}

func TestNoTrailingSpaces(t *testing.T) {
	got := plain(func(b *bytes.Buffer) {
		PrintTitle(b, "Seed Entries", 1)
		PrintList(b, []DescriptionItem{{"/etc/a.conf", "copy"}})
		PrintHints(b, Hint{"ws seed ls", "List"})
	})

	for _, line := range strings.Split(got, "\n") {
		assert.Assert(t, !strings.HasSuffix(line, " "), "trailing space in %q", line)
	}
}

func TestFormat(t *testing.T) {
	for in, want := range map[uint64]string{0: "0", 999: "999", 1000: "1,000", 524288: "524,288", 1234567: "1,234,567"} {
		assert.Equal(t, FormatCount(in), want)
	}

	for in, want := range map[time.Duration]string{
		30 * time.Second:              "less than a minute",
		5 * time.Minute:               "5 minutes",
		23*time.Hour + 59*time.Minute: "23 hours, 59 minutes",
		24*time.Hour + time.Minute:    "1 day, 1 minute",
		50 * time.Hour:                "2 days, 2 hours",
	} {
		assert.Equal(t, FormatDuration(in), want)
	}

	for in, want := range map[uint64]string{512: "512 B", 1536: "1.5 KiB", 3 << 30: "3.0 GiB"} {
		assert.Equal(t, FormatBytes(in), want)
	}
}
