package styles

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"gotest.tools/v3/assert"
)

func plain(fn func(*bytes.Buffer)) string {
	var buf bytes.Buffer
	fn(&buf)

	return ansi.Strip(buf.String())
}

func TestPrintTitleKeepsCase(t *testing.T) {
	assert.Equal(t, plain(func(b *bytes.Buffer) { PrintTitle(b, "Template ruff") }), "\n  Template ruff\n\n")
	assert.Equal(t, plain(func(b *bytes.Buffer) { PrintTitleCount(b, "gh", 3) }), "\n  gh (3)\n\n")
}

func TestPrintKeyValuesAligns(t *testing.T) {
	got := plain(func(b *bytes.Buffer) {
		PrintKeyValues(b, Pair{"URL", "http://localhost:8080"}, Pair{"Root", "/workspace"}, Pair{"Value", "line one\nline two"})
	})

	assert.Equal(t, got, "  URL    http://localhost:8080\n  Root   /workspace\n  Value  line one\n         line two\n")
}

func TestPrintListAligns(t *testing.T) {
	got := plain(func(b *bytes.Buffer) {
		PrintList(b, []DescriptionItem{{"gh", "Install gh CLI"}, {"terraform", "Install Terraform"}})
	})

	assert.Equal(t, got, "  • gh        — Install gh CLI\n  • terraform — Install Terraform\n")
}

func TestPrintBullets(t *testing.T) {
	assert.Equal(t, plain(func(b *bytes.Buffer) { PrintBullets(b, []string{"a", "b"}) }), "  • a\n  • b\n")
}

func TestPrintHintsAligns(t *testing.T) {
	got := plain(func(b *bytes.Buffer) {
		PrintHints(b, Hint{"ws feature install <name>", "Install a feature"}, Hint{"ws feature info <name>", "View feature details"})
	})

	assert.Equal(t, got, "\n  Quick actions\n    ws feature install <name>  Install a feature\n    ws feature info <name>     View feature details\n")
}

func TestStatusLines(t *testing.T) {
	assert.Equal(t, plain(func(b *bytes.Buffer) { PrintEmpty(b, "No open editors") }), "  ○ No open editors\n")
	assert.Equal(t, plain(func(b *bytes.Buffer) { PrintSuccess(b, "Done") }), "✓ Done\n")
	assert.Equal(t, plain(func(b *bytes.Buffer) { PrintWarning(b, "Careful") }), "⚠ Careful\n")
}

func TestPrintTextIndentsEveryLine(t *testing.T) {
	assert.Equal(t, plain(func(b *bytes.Buffer) { PrintText(b, "a\n\nb") }), "  a\n\n  b\n")
}

func TestFormatCount(t *testing.T) {
	for in, want := range map[uint64]string{0: "0", 999: "999", 1000: "1,000", 524288: "524,288", 1234567: "1,234,567"} {
		assert.Equal(t, FormatCount(in), want)
	}
}

func TestNoTrailingSpaces(t *testing.T) {
	got := plain(func(b *bytes.Buffer) {
		PrintTitleCount(b, "Seed Entries", 1)
		PrintList(b, []DescriptionItem{{"/etc/a.conf", "copy"}})
		PrintHints(b, Hint{"ws seed ls", "List"})
	})

	for _, line := range bytes.Split([]byte(got), []byte("\n")) {
		assert.Assert(t, !bytes.HasSuffix(line, []byte(" ")), "trailing space in %q", line)
	}
}
