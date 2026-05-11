package styles

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

var _ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func _stripANSI(s string) string {
	return _ansiRE.ReplaceAllString(s, "")
}

func TestRenderMarkdown_RendersTextContent(t *testing.T) {
	var buf bytes.Buffer
	err := RenderMarkdown(&buf, "Accepts a **space-delimited** package list.")
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(_stripANSI(buf.String()), "space-delimited"))
}

func TestRenderMarkdown_EmptyInputProducesNothing(t *testing.T) {
	var buf bytes.Buffer
	err := RenderMarkdown(&buf, "")
	assert.NilError(t, err)
	assert.Equal(t, "", buf.String())
}

func TestRenderMarkdown_WhitespaceOnlyProducesNothing(t *testing.T) {
	var buf bytes.Buffer
	err := RenderMarkdown(&buf, "   \n\n")
	assert.NilError(t, err)
	assert.Equal(t, "", buf.String())
}
