package net_test

import (
	"encoding/json"
	"io"
	gonet "net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/kloudkit/ws-cli/internals/net"
	"gotest.tools/v3/assert"
)

func TestSessionSocketPath(t *testing.T) {
	cases := []struct {
		name string
		home string
		xdg  string
		want string
	}{
		{"home default", "/home/kloud", "", "/home/kloud/.local/share/ws-server/session.sock"},
		{"absolute xdg", "/home/kloud", "/data", "/data/ws-server/session.sock"},
		{"relative xdg falls back", "/home/kloud", "rel/share", "/home/kloud/.local/share/ws-server/session.sock"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("HOME", c.home)
			t.Setenv("XDG_DATA_HOME", c.xdg)

			assert.Equal(t, net.SessionSocketPath(), c.want)
		})
	}
}

func listen(t *testing.T, socket string, handler http.Handler) {
	t.Helper()

	listener, err := gonet.Listen("unix", socket)
	assert.NilError(t, err)

	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()

	t.Cleanup(func() { _ = server.Close() })
}

func TestSendEnvelopeResolvesViaRegistry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("VSCODE_IPC_HOOK_CLI", "")

	pipe := filepath.Join(home, "pipe.sock")

	var resolvedFilePath string
	assert.NilError(t, os.MkdirAll(filepath.Dir(net.SessionSocketPath()), 0o755))

	listen(t, net.SessionSocketPath(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/session")
		resolvedFilePath = r.URL.Query().Get("filePath")
		_ = json.NewEncoder(w).Encode(map[string]string{"socketPath": pipe})
	}))

	var gotType string
	listen(t, pipe, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var envelope map[string]any
		_ = json.Unmarshal(body, &envelope)
		gotType, _ = envelope["type"].(string)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	got, err := net.SendEnvelope("/workspace/a.go", map[string]any{"type": "editorList"})
	assert.NilError(t, err)
	assert.Equal(t, string(got), `{"ok":true}`)
	assert.Equal(t, resolvedFilePath, "/workspace/a.go")
	assert.Equal(t, gotType, "editorList")
}

func TestSendEnvelopePrefersHook(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pipe := filepath.Join(t.TempDir(), "hook.sock")
	t.Setenv("VSCODE_IPC_HOOK_CLI", pipe)

	listen(t, pipe, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("[]"))
	}))

	got, err := net.SendEnvelope("", map[string]any{"type": "editorList"})
	assert.NilError(t, err)
	assert.Equal(t, string(got), "[]")
}

func TestSendEnvelopeNoEditorIsFriendly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("VSCODE_IPC_HOOK_CLI", "")

	_, err := net.SendEnvelope("", map[string]any{"type": "editorList"})
	assert.ErrorContains(t, err, "workspace editor")
}
