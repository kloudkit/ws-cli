package editor_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

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

func fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	assert.NilError(t, err)

	return data
}

func TestFetchDiagnosticsForwardsURI(t *testing.T) {
	body := fixture(t, "diagnostics.json")

	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		envelope := envelopeOf(t, r)
		assert.Equal(t, envelope["type"], "diagnostics")
		assert.Equal(t, envelope["uri"], "file:///workspace/main.go")
		_, _ = w.Write(body)
	}))

	got, err := editor.FetchDiagnostics("file:///workspace/main.go")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, body)
}

func TestFetchDiagnosticsWithoutURIOmitsField(t *testing.T) {
	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		envelope := envelopeOf(t, r)
		_, hasURI := envelope["uri"]
		assert.Assert(t, !hasURI)
		_, _ = w.Write([]byte("[]"))
	}))

	got, err := editor.FetchDiagnostics("")
	assert.NilError(t, err)
	assert.Equal(t, string(got), "[]")
}

func TestFetchEditors(t *testing.T) {
	body := fixture(t, "editors.json")

	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, envelopeOf(t, r)["type"], "editorList")
		_, _ = w.Write(body)
	}))

	got, err := editor.FetchEditors()
	assert.NilError(t, err)
	assert.DeepEqual(t, got, body)
}

func TestFetchSelection(t *testing.T) {
	body := fixture(t, "selection.json")

	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, envelopeOf(t, r)["type"], "editorSelection")
		_, _ = w.Write(body)
	}))

	got, err := editor.FetchSelection()
	assert.NilError(t, err)
	assert.DeepEqual(t, got, body)
}

func TestFetchSelectionNullIsNil(t *testing.T) {
	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("null"))
	}))

	got, err := editor.FetchSelection()
	assert.NilError(t, err)
	assert.Equal(t, len(got), 0)
}

func TestOpenSendsEnvelope(t *testing.T) {
	var envelope map[string]any

	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		envelope = envelopeOf(t, r)
		_, _ = w.Write(fixture(t, "open.json"))
	}))

	err := editor.Open(editor.OpenRequest{
		Path:      "/workspace/main.go",
		NewWindow: true,
		Selection: &editor.Range{End: editor.Position{Line: 2, Character: 5}},
	})
	assert.NilError(t, err)

	assert.Equal(t, envelope["type"], "editorOpen")
	assert.Equal(t, envelope["path"], "/workspace/main.go")
	assert.Equal(t, envelope["newWindow"], true)
	assert.Assert(t, envelope["selection"] != nil)
}

func TestNotifyForwardsPayload(t *testing.T) {
	var envelope map[string]any

	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		envelope = envelopeOf(t, r)
		_, _ = w.Write([]byte(`{"action":"Yes","timedOut":false}`))
	}))

	body, err := editor.Notify(editor.NotifyRequest{
		Message: "Deploy now?",
		Actions: []any{"Yes", "No"},
		Timeout: 5000,
	})
	assert.NilError(t, err)

	assert.Equal(t, envelope["type"], "notify")
	assert.Equal(t, envelope["message"], "Deploy now?")
	assert.Assert(t, envelope["actions"] != nil)
	assert.Equal(t, envelope["timeout"], float64(5000))
	assert.Equal(t, string(body), `{"action":"Yes","timedOut":false}`)
}

func TestNotifyOmitsEmptyOptionals(t *testing.T) {
	var envelope map[string]any

	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		envelope = envelopeOf(t, r)
		_, _ = w.Write([]byte("null"))
	}))

	_, err := editor.Notify(editor.NotifyRequest{Message: "hi"})
	assert.NilError(t, err)

	_, hasDetail := envelope["detail"]
	_, hasActions := envelope["actions"]
	_, hasTimeout := envelope["timeout"]
	assert.Assert(t, !hasDetail)
	assert.Assert(t, !hasActions)
	assert.Assert(t, !hasTimeout)
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

func TestServerDownIsFriendly(t *testing.T) {
	t.Setenv("VSCODE_IPC_HOOK_CLI", filepath.Join(t.TempDir(), "absent.sock"))

	_, err := editor.FetchEditors()
	assert.ErrorContains(t, err, "workspace editor")
}

func TestFixturesMatchContract(t *testing.T) {
	t.Run("diagnostics", func(t *testing.T) {
		var v []editor.DiagnosticFile
		decodeStrict(t, "diagnostics.json", &v)
	})

	t.Run("editors", func(t *testing.T) {
		var v []editor.Tab
		decodeStrict(t, "editors.json", &v)
	})

	t.Run("selection", func(t *testing.T) {
		var v editor.Selection
		decodeStrict(t, "selection.json", &v)
	})

	t.Run("open", func(t *testing.T) {
		var v struct {
			Opened bool `json:"opened"`
		}
		decodeStrict(t, "open.json", &v)
	})
}

func decodeStrict(t *testing.T, name string, target any) {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(fixture(t, name)))
	decoder.DisallowUnknownFields()
	assert.NilError(t, decoder.Decode(target))
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

func TestDecodeTabsNormalizesPaths(t *testing.T) {
	tabs, err := editor.DecodeTabs([]byte(`[{"path":"\\workspace\\go.work","active":true},{"path":"/etc/hosts"}]`))
	assert.NilError(t, err)

	assert.Equal(t, len(tabs), 2)
	assert.Equal(t, tabs[0].Path, "/workspace/go.work")
	assert.Equal(t, tabs[1].Path, "/etc/hosts")
}

func TestDecodeSelectionNormalizesPath(t *testing.T) {
	selection, err := editor.DecodeSelection([]byte(`{"path":"\\workspace\\main.go","text":"x"}`))
	assert.NilError(t, err)

	assert.Equal(t, selection.Path, "/workspace/main.go")
	assert.Equal(t, selection.Text, "x")
}
