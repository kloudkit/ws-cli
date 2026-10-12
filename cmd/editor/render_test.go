package editor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
)

func render(t *testing.T, fn func(*bytes.Buffer) error) string {
	t.Helper()

	var out bytes.Buffer
	assert.NilError(t, fn(&out))

	return testutil.StripAnsi(out.String())
}

func TestRenderEditors(t *testing.T) {
	t.Run("Tabs", func(t *testing.T) {
		body := []byte(`[
			{"path":"\\workspace\\go.work","languageId":"go.work","active":true,"dirty":false},
			{"path":"/workspace/README.md","languageId":null,"active":false,"dirty":true}
		]`)

		got := render(t, func(out *bytes.Buffer) error { return renderEditors(out, body) })

		assert.Assert(t, strings.Contains(strings.ToLower(got), "open editors"))
		assert.Assert(t, strings.Contains(got, "(2)"))
		assert.Assert(t, strings.Contains(got, "/workspace/go.work"))
		assert.Assert(t, !strings.Contains(got, `\workspace`))
		assert.Assert(t, strings.Contains(got, "/workspace/README.md"))
		assert.Assert(t, strings.Contains(got, "go.work"))
		assert.Equal(t, strings.Count(got, "✓"), 2)
	})

	t.Run("Empty", func(t *testing.T) {
		got := render(t, func(out *bytes.Buffer) error { return renderEditors(out, []byte(`[]`)) })

		assert.Assert(t, strings.Contains(got, "No open editors"))
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		err := renderEditors(&bytes.Buffer{}, []byte(`{`))

		assert.ErrorContains(t, err, "error parsing editor response")
	})
}

func TestRenderSelection(t *testing.T) {
	t.Run("Selection", func(t *testing.T) {
		body := []byte(`{"path":"\\workspace\\main.go","text":"func main()","range":{"start":{"line":11,"character":0},"end":{"line":11,"character":11}}}`)

		got := render(t, func(out *bytes.Buffer) error { return renderSelection(out, body) })

		assert.Assert(t, strings.Contains(strings.ToLower(got), "selection"))
		assert.Assert(t, strings.Contains(got, "/workspace/main.go"))
		assert.Assert(t, strings.Contains(got, "12:1-12:12"))
		assert.Assert(t, strings.Contains(got, "func main()"))
	})

	t.Run("Empty", func(t *testing.T) {
		got := render(t, func(out *bytes.Buffer) error { return renderSelection(out, nil) })

		assert.Assert(t, strings.Contains(got, "No active selection"))
	})
}

func TestRenderDiagnostics(t *testing.T) {
	t.Run("Diagnostics", func(t *testing.T) {
		body := []byte(`[
			{"uri":"file:///workspace/main.go","items":[
				{"severity":{"code":0,"label":"Error"},"message":"undefined: foo","source":"compiler","range":{"start":{"line":4,"character":1},"end":{"line":4,"character":4}}},
				{"severity":{"code":1,"label":"Warning"},"message":"unused","source":"vet","range":{"start":{"line":9,"character":0},"end":{"line":9,"character":1}}}
			]},
			{"uri":"file:///workspace/clean.go","items":[]}
		]`)

		got := render(t, func(out *bytes.Buffer) error { return renderDiagnostics(out, body) })

		assert.Assert(t, strings.Contains(strings.ToLower(got), "diagnostics"))
		assert.Assert(t, strings.Contains(got, "(2)"))
		assert.Assert(t, strings.Contains(got, "file:///workspace/main.go"))
		assert.Assert(t, !strings.Contains(got, "clean.go"))
		assert.Assert(t, strings.Contains(got, "5:2"))
		assert.Assert(t, strings.Contains(got, "undefined: foo"))
		assert.Assert(t, strings.Contains(got, "Warning"))
	})

	t.Run("Empty", func(t *testing.T) {
		got := render(t, func(out *bytes.Buffer) error {
			return renderDiagnostics(out, []byte(`[{"uri":"file:///a","items":[]}]`))
		})

		assert.Assert(t, strings.Contains(got, "No diagnostics"))
	})
}
