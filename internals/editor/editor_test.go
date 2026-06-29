package editor_test

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/editor"
	"gotest.tools/v3/assert"
)

func startStub(t *testing.T, handler http.Handler) {
	t.Helper()

	socket := filepath.Join(t.TempDir(), "ws-ipc.sock")
	t.Setenv(config.EnvIPCSocket, socket)

	listener, err := net.Listen("unix", socket)
	assert.NilError(t, err)

	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()

	t.Cleanup(func() { _ = server.Close() })
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	assert.NilError(t, err)

	return data
}

func TestFetchDiagnosticsForwardsURI(t *testing.T) {
	body := fixture(t, "diagnostics.json")

	startStub(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodGet)
		assert.Equal(t, r.URL.Path, "/diagnostics")
		assert.Equal(t, r.URL.Query().Get("uri"), "file:///workspace/main.go")
		_, _ = w.Write(body)
	}))

	got, err := editor.FetchDiagnostics("file:///workspace/main.go")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, body)
}

func TestFetchDiagnosticsWithoutURIOmitsQuery(t *testing.T) {
	startStub(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.RawQuery, "")
		_, _ = w.Write([]byte("[]"))
	}))

	got, err := editor.FetchDiagnostics("")
	assert.NilError(t, err)
	assert.Equal(t, string(got), "[]")
}

func TestFetchEditors(t *testing.T) {
	body := fixture(t, "editors.json")

	startStub(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/editors")
		_, _ = w.Write(body)
	}))

	got, err := editor.FetchEditors()
	assert.NilError(t, err)
	assert.DeepEqual(t, got, body)
}

func TestFetchSelection(t *testing.T) {
	body := fixture(t, "selection.json")

	startStub(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))

	got, err := editor.FetchSelection()
	assert.NilError(t, err)
	assert.DeepEqual(t, got, body)
}

func TestFetchSelectionNoContentIsNil(t *testing.T) {
	startStub(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	got, err := editor.FetchSelection()
	assert.NilError(t, err)
	assert.Equal(t, len(got), 0)
}

func TestOpenSendsRequestBody(t *testing.T) {
	var got editor.OpenRequest

	startStub(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodPost)
		assert.Equal(t, r.URL.Path, "/open")
		assert.NilError(t, json.NewDecoder(r.Body).Decode(&got))
		_, _ = w.Write(fixture(t, "open.json"))
	}))

	err := editor.Open(editor.OpenRequest{
		Path:      "/workspace/main.go",
		Window:    "new",
		Selection: &editor.Range{End: editor.Position{Line: 2, Character: 5}},
	})
	assert.NilError(t, err)

	assert.Equal(t, got.Path, "/workspace/main.go")
	assert.Equal(t, got.Window, "new")
	assert.Assert(t, got.Selection != nil)
	assert.Equal(t, got.Selection.End.Line, 2)
}

func TestErrorResponseSurfacesBody(t *testing.T) {
	startStub(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("kernel exploded"))
	}))

	_, err := editor.FetchEditors()
	assert.ErrorContains(t, err, "500")
	assert.ErrorContains(t, err, "kernel exploded")
}

func TestServerDownIsFriendly(t *testing.T) {
	t.Setenv(config.EnvIPCSocket, filepath.Join(t.TempDir(), "absent.sock"))

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
