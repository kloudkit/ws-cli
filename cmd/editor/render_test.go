package editor

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
)

func TestRender(t *testing.T) {
	cases := []struct {
		name   string
		render func(io.Writer, []byte) error
		body   string
		has    []string
		lacks  []string
		checks int
	}{
		{
			name: "Editors", render: renderEditors, checks: 2,
			body: `[{"path":"\\workspace\\go.work","languageId":"go.work","active":true,"dirty":false},
				{"path":"/workspace/README.md","languageId":null,"active":false,"dirty":true}]`,
			has:   []string{"Open Editors", "─ 2\n", "/workspace/go.work", "/workspace/README.md", "go.work"},
			lacks: []string{`\workspace`},
		},
		{name: "EditorsEmpty", render: renderEditors, body: `[]`, has: []string{"No open editors"}},
		{
			name: "Selection", render: renderSelection,
			body: `{"path":"\\workspace\\main.go","text":"func main()","range":{"start":{"line":11,"character":0},"end":{"line":11,"character":11}}}`,
			has:  []string{"Selection", "/workspace/main.go", "12:1-12:12", "func main()"},
		},
		{name: "SelectionEmpty", render: renderSelection, has: []string{"No active selection"}},
		{
			name: "Diagnostics", render: renderDiagnostics,
			body: `[{"uri":"file:///workspace/main.go","items":[
				{"severity":{"code":0,"label":"Error"},"message":"undefined: foo","source":"compiler","range":{"start":{"line":4,"character":1},"end":{"line":4,"character":4}}},
				{"severity":{"code":1,"label":"Warning"},"message":"unused","source":"vet","range":{"start":{"line":9,"character":0},"end":{"line":9,"character":1}}}]},
				{"uri":"file:///workspace/clean.go","items":[]}]`,
			has:   []string{"Diagnostics", "─ 2\n", "file:///workspace/main.go", "5:2", "undefined: foo", "Warning"},
			lacks: []string{"clean.go"},
		},
		{name: "DiagnosticsEmpty", render: renderDiagnostics, body: `[{"uri":"file:///a","items":[]}]`, has: []string{"No diagnostics"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buffer bytes.Buffer
			var body []byte
			if c.body != "" {
				body = []byte(c.body)
			}
			assert.NilError(t, c.render(&buffer, body))
			out := testutil.StripAnsi(buffer.String())

			for _, want := range c.has {
				assert.Assert(t, strings.Contains(out, want), "want %q in %q", want, out)
			}
			for _, want := range c.lacks {
				assert.Assert(t, !strings.Contains(out, want), "want no %q in %q", want, out)
			}
			if c.checks > 0 {
				assert.Equal(t, strings.Count(out, "✓"), c.checks)
			}
		})
	}

	for name, render := range map[string]func(io.Writer, []byte) error{"editor": renderEditors, "selection": renderSelection, "diagnostics": renderDiagnostics} {
		assert.ErrorContains(t, render(io.Discard, []byte(`{`)), "error parsing "+name+" response")
	}
}
