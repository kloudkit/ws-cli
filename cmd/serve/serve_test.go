package serve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
)

func TestServeTakenPort(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)
	defer taken.Close()

	port := strconv.Itoa(taken.Addr().(*net.TCPAddr).Port)

	for _, args := range [][]string{{"current"}, {"metrics", "--collectors", "workspace"}, {"metrics", "--collectors", "bogus,workspace"}} {
		testutil.ResetCommandFlags(ServeCmd)

		var out bytes.Buffer
		ServeCmd.SetOut(&out)
		ServeCmd.SetErr(&out)
		ServeCmd.SetArgs(append(args, "--bind", "127.0.0.1", "--port", port))

		assert.ErrorContains(t, ServeCmd.Execute(), "address already in use", args[0])
		assert.Assert(t, !strings.Contains(out.String(), "Serving"))
		assert.Equal(t, strings.Contains(out.String(), "Unknown collector 'bogus'"), strings.Contains(args[len(args)-1], "bogus"))
	}
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestServeAnnounces(t *testing.T) {
	for _, args := range [][]string{{"current"}, {"metrics", "--collectors", "workspace"}} {
		out := &syncBuffer{}
		testutil.ResetCommandFlags(ServeCmd)
		ServeCmd.SetOut(out)
		ServeCmd.SetArgs(append(args, "--bind", "127.0.0.1", "--port", "0"))

		go func() { _ = ServeCmd.Execute() }()

		assert.Assert(t, poll(func() bool { return strings.Contains(out.String(), "Press Ctrl+C") }), out.String())
		assert.Assert(t, strings.Contains(testutil.StripAnsi(out.String()), "URL"))
	}
}

func poll(cond func() bool) bool {
	for range 200 {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func shareSocket(t *testing.T, unshare <-chan struct{}, refuse string) {
	t.Helper()

	testutil.ServeSession(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if refuse != "" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"error":"`+refuse+`"}`)
			return
		}

		var body struct {
			Port int `json:"port"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = fmt.Fprintf(w, `{"event":"shared","port":%d,"access":"public","urls":["https://%d.ws.example"],"loginRequired":true}`+"\n", body.Port, body.Port)
		w.(http.Flusher).Flush()

		select {
		case <-unshare:
			_, _ = io.WriteString(w, `{"event":"unshared","by":"the editor"}`+"\n")
		case <-r.Context().Done():
		}
	}))
}

func TestServeShareConfinesAndEnds(t *testing.T) {
	root := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(root, "index.html"), []byte("home"), 0o644))
	assert.NilError(t, os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=x"), 0o644))
	assert.NilError(t, os.Mkdir(filepath.Join(root, "empty"), 0o755))
	t.Chdir(root)

	unshare := make(chan struct{})
	shareSocket(t, unshare, "")

	out := &syncBuffer{}
	testutil.ResetCommandFlags(ServeCmd)
	ServeCmd.SetOut(out)
	ServeCmd.SetErr(out)
	ServeCmd.SetArgs([]string{"current", "--share", "--bind", "127.0.0.1", "--port", "0"})

	finished := make(chan error, 1)
	go func() { finished <- ServeCmd.Execute() }()

	assert.Assert(t, poll(func() bool { return strings.Contains(out.String(), "Press Ctrl+C") }), out.String())

	text := testutil.StripAnsi(out.String())
	local := regexp.MustCompile(`http://localhost:\d+`).FindString(text)
	assert.Assert(t, local != "", text)
	assert.Assert(t, strings.Contains(text, ".ws.example"), text)

	for path, status := range map[string]int{"/": http.StatusOK, "/.env": http.StatusNotFound, "/empty/": http.StatusNotFound} {
		resp, err := http.Get(local + path)
		assert.NilError(t, err)
		resp.Body.Close()
		assert.Equal(t, resp.StatusCode, status, path)
	}

	close(unshare)

	select {
	case err := <-finished:
		assert.NilError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("serve current --share kept serving after the share ended")
	}

	assert.Assert(t, strings.Contains(testutil.StripAnsi(out.String()), "unshared by the editor"))

	_, err := http.Get(local + "/")
	assert.Assert(t, err != nil, "the listener is still open")
}

func TestServeShareRefused(t *testing.T) {
	t.Chdir(t.TempDir())
	shareSocket(t, nil, "public ports are off; set WS_SERVER_PROXY_ALLOW_PUBLIC=true to allow them")

	testutil.ResetCommandFlags(ServeCmd)
	out := &syncBuffer{}
	ServeCmd.SetOut(out)
	ServeCmd.SetErr(out)
	ServeCmd.SetArgs([]string{"current", "--share", "--bind", "127.0.0.1", "--port", "0"})

	assert.ErrorContains(t, ServeCmd.Execute(), "WS_SERVER_PROXY_ALLOW_PUBLIC")
	assert.Assert(t, !strings.Contains(out.String(), "Press Ctrl+C"))
}
