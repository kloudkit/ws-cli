package editor_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kloudkit/ws-cli/internals/editor"
	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
)

func envelopeOf(t *testing.T, r *http.Request) map[string]any {
	t.Helper()

	assert.Equal(t, r.Method, http.MethodPost)

	body, err := io.ReadAll(r.Body)
	assert.NilError(t, err)

	var envelope map[string]any
	assert.NilError(t, json.Unmarshal(body, &envelope))

	return envelope
}

func TestEnvelopes(t *testing.T) {
	cases := []struct {
		name     string
		response string
		call     func() ([]byte, error)
		want     map[string]any
		absent   []string
		body     string
	}{
		{"DiagnosticsURI", `[{"uri":"x"}]`, func() ([]byte, error) { return editor.FetchDiagnostics("file:///workspace/main.go") },
			map[string]any{"type": "diagnostics", "uri": "file:///workspace/main.go"}, nil, `[{"uri":"x"}]`},
		{"DiagnosticsAll", "[]", func() ([]byte, error) { return editor.FetchDiagnostics("") },
			map[string]any{"type": "diagnostics"}, []string{"uri"}, "[]"},
		{"Editors", "[]", editor.FetchEditors, map[string]any{"type": "editorList"}, nil, "[]"},
		{"Selection", `{"path":"a"}`, editor.FetchSelection, map[string]any{"type": "editorSelection"}, nil, `{"path":"a"}`},
		{"SelectionNull", "null", editor.FetchSelection, map[string]any{"type": "editorSelection"}, nil, ""},
		{"Open", `{"opened":true}`, func() ([]byte, error) {
			return nil, editor.Open(editor.OpenRequest{Path: "/workspace/main.go", NewWindow: true, Selection: &editor.Range{End: editor.Position{Line: 2, Character: 5}}})
		}, map[string]any{"type": "editorOpen", "path": "/workspace/main.go", "newWindow": true, "selection": map[string]any{
			"start": map[string]any{"line": float64(0), "character": float64(0)},
			"end":   map[string]any{"line": float64(2), "character": float64(5)},
		}}, nil, ""},
		{"OpenTab", `{"opened":true}`, func() ([]byte, error) { return nil, editor.Open(editor.OpenRequest{Path: "/workspace/a.md"}) },
			map[string]any{"type": "editorOpen", "newWindow": false, "preview": false}, []string{"modal", "selection"}, ""},
		{"OpenModal", `{"opened":true}`, func() ([]byte, error) {
			return nil, editor.Open(editor.OpenRequest{Path: "/workspace/a.md", Modal: true})
		},
			map[string]any{"type": "editorOpen", "path": "/workspace/a.md", "modal": true}, nil, ""},
		{"Reveal", "null", func() ([]byte, error) { return nil, editor.Reveal("/workspace/src") },
			map[string]any{"type": "editorReveal", "path": "/workspace/src"}, nil, ""},
		{"Terminal", "null", func() ([]byte, error) {
			return nil, editor.OpenTerminal(editor.TerminalRequest{Cwd: "/workspace", Name: "tests", Command: "go test ./..."})
		}, map[string]any{"type": "editorTerminal", "cwd": "/workspace", "name": "tests", "command": "go test ./..."}, nil, ""},
		{"TerminalMinimal", "null", func() ([]byte, error) { return nil, editor.OpenTerminal(editor.TerminalRequest{Cwd: "/workspace"}) },
			map[string]any{"type": "editorTerminal", "cwd": "/workspace"}, []string{"name", "command"}, ""},
		{"Download", `{"downloaded":true}`, func() ([]byte, error) { return editor.Download("/workspace/big.iso") },
			map[string]any{"type": "editorDownload", "path": "/workspace/big.iso"}, nil, `{"downloaded":true}`},
		{"Notify", `{"action":"Yes","timedOut":false}`, func() ([]byte, error) {
			return editor.Notify(editor.NotifyRequest{Message: "Deploy now?", Actions: []any{"Yes", "No"}, Timeout: new(5000)})
		}, map[string]any{"type": "notify", "message": "Deploy now?", "actions": []any{"Yes", "No"}, "timeout": float64(5000)}, nil, `{"action":"Yes","timedOut":false}`},
		{"NotifyMinimal", "null", func() ([]byte, error) { return editor.Notify(editor.NotifyRequest{Message: "hi"}) },
			map[string]any{"type": "notify", "message": "hi"}, []string{"detail", "actions", "timeout", "modal", "severity"}, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var envelope map[string]any

			testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				envelope = envelopeOf(t, r)
				_, _ = w.Write([]byte(c.response))
			}))

			body, err := c.call()
			assert.NilError(t, err)
			assert.Equal(t, string(body), c.body)

			for key, want := range c.want {
				assert.DeepEqual(t, envelope[key], want)
			}

			for _, key := range c.absent {
				_, has := envelope[key]
				assert.Assert(t, !has, key)
			}
		})
	}
}

func TestErrorResponseSurfacesBody(t *testing.T) {
	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("kernel exploded"))
	}))

	_, err := editor.FetchEditors()
	assert.ErrorContains(t, err, "500")
	assert.ErrorContains(t, err, "kernel exploded")
}

func TestUnknownRequestNeedsNewerImage(t *testing.T) {
	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`"Unknown message type: editorDownload"`))
	}))

	_, err := editor.Download("/workspace/a.txt")
	assert.ErrorContains(t, err, "newer workspace image")
}

func TestDownloadGivesUpWhenTheEditorNeverAnswers(t *testing.T) {
	assert.Equal(t, editor.DownloadWait, 310*time.Second)

	original := editor.DownloadWait
	editor.DownloadWait = 200 * time.Millisecond
	t.Cleanup(func() { editor.DownloadWait = original })

	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))

	_, err := editor.Download("/workspace/big.iso")
	assert.Error(t, err, "the workspace editor did not answer within 200ms")
}

func TestServerDownIsFriendly(t *testing.T) {
	t.Setenv("VSCODE_IPC_HOOK_CLI", filepath.Join(t.TempDir(), "absent.sock"))

	_, err := editor.FetchEditors()
	assert.ErrorContains(t, err, "workspace editor")
}

func TestFixturesMatchContract(t *testing.T) {
	for name, target := range map[string]any{
		"diagnostics.json": &[]editor.DiagnosticFile{},
		"editors.json":     &[]editor.Tab{},
		"selection.json":   &editor.Selection{},
		"open.json":        &struct{ Opened bool }{},
		"download.json":    &editor.DownloadResult{},
	} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		assert.NilError(t, err)

		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		assert.NilError(t, decoder.Decode(target), name)
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{`\workspace\go.work`, "/workspace/go.work"},
		{`\home\kloud\a b\c.txt`, "/home/kloud/a b/c.txt"},
		{"/workspace/go.work", "/workspace/go.work"},
		{`/workspace/odd\name`, `/workspace/odd\name`},
		{"", ""},
	}

	for _, tt := range tests {
		assert.Equal(t, editor.NormalizePath(tt.in), tt.want)
	}
}

func TestDecodeNormalizesPaths(t *testing.T) {
	tabs, err := editor.DecodeTabs([]byte(`[{"path":"\\workspace\\go.work","active":true},{"path":"/etc/hosts"}]`))
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{tabs[0].Path, tabs[1].Path}, []string{"/workspace/go.work", "/etc/hosts"})

	selection, err := editor.DecodeSelection([]byte(`{"path":"\\workspace\\main.go","text":"x"}`))
	assert.NilError(t, err)
	assert.DeepEqual(t, *selection, editor.Selection{Path: "/workspace/main.go", Text: "x"})
}
