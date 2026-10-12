package styles

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
)

func TestRenderMarkdown_RendersTextContent(t *testing.T) {
	var buf bytes.Buffer
	err := RenderMarkdown(&buf, "Accepts a **space-delimited** package list.")
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(testutil.StripAnsi(buf.String()), "space-delimited"))
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
