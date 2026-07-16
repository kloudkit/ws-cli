package clipboard_test

import (
	"bytes"
	"encoding/json"
	"io"
	gonet "net"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/kloudkit/ws-cli/internals/clipboard"
	"gotest.tools/v3/assert"
)

func startPipe(t *testing.T, handler http.Handler) {
	t.Helper()

	socket := filepath.Join(t.TempDir(), "pipe.sock")
	t.Setenv("VSCODE_IPC_HOOK_CLI", socket)

	listener, err := gonet.Listen("unix", socket)
	assert.NilError(t, err)

	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()

	t.Cleanup(func() { _ = server.Close() })
}

func TestPasteWritesDecodedString(t *testing.T) {
	startPipe(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var envelope map[string]any
		_ = json.Unmarshal(body, &envelope)
		assert.Equal(t, envelope["type"], "clipboardRead")

		_, _ = w.Write([]byte(`"line one\nline two"`))
	}))

	var out bytes.Buffer
	assert.NilError(t, clipboard.Paste(&out))
	assert.Equal(t, out.String(), "line one\nline two")
}

func TestPasteNullIsEmpty(t *testing.T) {
	startPipe(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("null"))
	}))

	var out bytes.Buffer
	assert.NilError(t, clipboard.Paste(&out))
	assert.Equal(t, out.String(), "")
}
