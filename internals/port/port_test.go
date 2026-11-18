package port_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	wsnet "github.com/kloudkit/ws-cli/internals/net"
	"github.com/kloudkit/ws-cli/internals/port"
	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
)

const procNetTCP = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1F40 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 1 1 0000000000000000 100 0 0 10 0
   1: 00000000:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 2 1 0000000000000000 100 0 0 10 0
   2: 0100007F:1F40 0100007F:D431 01 00000000:00000000 00:00000000 00000000  1000        0 3 1 0000000000000000 20 4 30 10 -1
`

const procNetTCP6 = `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000000000000:1F40 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 4 1 0000000000000000 100 0 0 10 0
   1: 00000000000000000000000001000000:0BB8 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 5 1 0000000000000000 100 0 0 10 0
`

func writeFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	assert.NilError(t, os.WriteFile(path, []byte(content), 0o644))

	return path
}

func TestListenersReadsOnlyListeningSocketsFromBothTables(t *testing.T) {
	listeners, err := port.Listeners(writeFile(t, "tcp", procNetTCP), writeFile(t, "tcp6", procNetTCP6))

	assert.NilError(t, err)
	assert.DeepEqual(t, listeners, []port.Listener{
		{Port: 3000, Addresses: []string{"::1"}},
		{Port: 8000, Addresses: []string{"127.0.0.1", "::"}},
		{Port: 8080, Addresses: []string{"0.0.0.0"}},
	})
}

func TestListenersFailsWhenNoTableIsReadable(t *testing.T) {
	_, err := port.Listeners(filepath.Join(t.TempDir(), "missing"))

	assert.ErrorContains(t, err, "/proc/net/tcp")
}

func TestParsePort(t *testing.T) {
	for _, raw := range []string{"0", "65536", "http", "-1", ""} {
		_, err := port.ParsePort(raw)
		assert.ErrorContains(t, err, "invalid port", raw)
	}

	number, err := port.ParsePort("8000")
	assert.NilError(t, err)
	assert.Equal(t, number, 8000)
}

type fakeServer struct {
	unshare chan string
	closed  chan struct{}
	body    map[string]any
}

func sessionSocket(t *testing.T, handler http.Handler) {
	t.Helper()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", "")
	assert.NilError(t, os.MkdirAll(filepath.Dir(wsnet.SessionSocketPath()), 0o755))
	testutil.ServeUnix(t, wsnet.SessionSocketPath(), handler)
}

func holdingServer(t *testing.T, first string) *fakeServer {
	t.Helper()

	fake := &fakeServer{unshare: make(chan string, 1), closed: make(chan struct{})}

	sessionSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/ports/share")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &fake.body)

		w.Header().Set("content-type", "application/x-ndjson")
		_, _ = io.WriteString(w, first+"\n")
		w.(http.Flusher).Flush()

		select {
		case by := <-fake.unshare:
			_, _ = io.WriteString(w, `{"event":"unshared","port":8000,"by":"`+by+`"}`+"\n")
		case <-r.Context().Done():
			close(fake.closed)
		}
	}))

	return fake
}

const sharedEvent = `{"event":"shared","port":8000,"access":"public","source":{"kind":"cli","pid":1},"since":"2026-11-18T00:00:00Z","urls":["https://8000.ws.example"],"loginRequired":true}`

func TestHoldAnnouncesTheURLsAndReleasesWhenStopped(t *testing.T) {
	fake := holdingServer(t, sharedEvent)
	ctx, cancel := context.WithCancel(context.Background())

	var announced port.Shared
	done := make(chan struct{})

	go func() {
		defer close(done)
		ended, err := port.Hold(ctx, 8000, func(shared port.Shared) {
			announced = shared
			cancel()
		})
		assert.NilError(t, err)
		assert.Assert(t, ended == nil)
	}()

	<-done

	assert.DeepEqual(t, announced.URLs, []string{"https://8000.ws.example"})
	assert.Equal(t, announced.LoginRequired, true)
	assert.Equal(t, fake.body["access"], "public")
	assert.DeepEqual(t, fake.body["source"], map[string]any{"kind": "cli", "pid": float64(os.Getpid())})

	select {
	case <-fake.closed:
	case <-time.After(3 * time.Second):
		t.Fatal("the holder never closed its request")
	}
}

func TestHoldEndsWhenUnsharedElsewhere(t *testing.T) {
	fake := holdingServer(t, sharedEvent)

	ended, err := port.Hold(context.Background(), 8000, func(port.Shared) { fake.unshare <- "the editor" })

	assert.NilError(t, err)
	assert.Equal(t, ended.By, "the editor")
}

func refusingServer(t *testing.T, status int, body string) {
	t.Helper()

	sessionSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
}

func TestHoldPassesARefusalThrough(t *testing.T) {
	for name, refusal := range map[string]string{
		"off":    "public ports are off; set WS_SERVER_PROXY_ALLOW_PUBLIC=true to allow them",
		"domain": "public ports need a proxy domain; set WS_SERVER_PROXY_DOMAIN",
		"own":    "port 8080 is the workspace server's own port and cannot be shared",
	} {
		t.Run(name, func(t *testing.T) {
			raw, _ := json.Marshal(map[string]string{"error": refusal})
			refusingServer(t, http.StatusForbidden, string(raw))

			_, err := port.Hold(context.Background(), 8000, func(port.Shared) { t.Fatal("announced a refused share") })

			assert.Error(t, err, refusal)
		})
	}
}

func TestAnOldServerNeedsANewerImage(t *testing.T) {
	refusingServer(t, http.StatusNotFound, `{"error":"not found"}`)

	_, holdErr := port.Hold(context.Background(), 8000, func(port.Shared) {})
	_, listErr := port.List()
	unshareErr := port.Unshare(8000)

	for _, err := range []error{holdErr, listErr, unshareErr} {
		assert.ErrorContains(t, err, "needs a newer workspace image")
	}
}

func TestNoServerSaysSo(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", "")

	_, err := port.List()

	assert.ErrorContains(t, err, "cannot reach the workspace server")
}

func TestUnshareSendsThePortAndThisProcess(t *testing.T) {
	var body map[string]any
	sessionSocket(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/ports/unshare")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))

	assert.NilError(t, port.Unshare(8000))
	assert.Equal(t, body["port"], float64(8000))
	assert.DeepEqual(t, body["source"], map[string]any{"kind": "cli", "pid": float64(os.Getpid())})
}

func TestTabAndClearUseTheEditorVerbs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var types []string
	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var envelope map[string]any
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &envelope)
		types = append(types, envelope["type"].(string))

		if envelope["type"] == "portList" {
			_, _ = io.WriteString(w, `[{"port":8000,"host":"localhost","privacy":"public","forwarded":true,"running":false,"process":null,"name":null}]`)
			return
		}
		_, _ = io.WriteString(w, `{"closed":[8000]}`)
	}))

	tab, err := port.Tab()
	assert.NilError(t, err)
	assert.Equal(t, tab[0].Privacy, "public")
	assert.Equal(t, *tab[0].Running, false)

	cleared, err := port.Clear()
	assert.NilError(t, err)
	assert.DeepEqual(t, cleared.Closed, []int{8000})
	assert.DeepEqual(t, types, []string{"portList", "portClear"})
}
