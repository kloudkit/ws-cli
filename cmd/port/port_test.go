package port

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/cmd/clitest"
	"github.com/kloudkit/ws-cli/internals/port"
	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
)

func sharedThenUnshared(loginRequired string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"event":"shared","port":8000,"access":"public","source":{"kind":"cli","pid":1},"urls":["https://8000.ws.example"],"loginRequired":`+loginRequired+"}\n")
		_, _ = io.WriteString(w, `{"event":"unshared","port":8000,"by":"ws-cli pid 77"}`+"\n")
	}
}

func TestSharePrintsTheURLsAndExitsWhenUnsharedElsewhere(t *testing.T) {
	testutil.ServeSession(t, sharedThenUnshared("true"))

	out, err := clitest.Run(PortCmd, "", "share", "8000")

	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(out, "https://8000.ws.example"), out)
	assert.Assert(t, strings.Contains(out, "Press Ctrl+C"), out)
	assert.Assert(t, strings.Contains(out, "Port 8000 is private again: unshared by ws-cli pid 77"), out)
	assert.Assert(t, !strings.Contains(out, "sign-in is off"), out)
}

func TestShareWarnsWhenSignInIsOff(t *testing.T) {
	testutil.ServeSession(t, sharedThenUnshared("false"))

	out, err := clitest.Run(PortCmd, "", "share", "8000")

	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(out, "every forwarded port is already open"), out)
}

func TestShareAndUnshareRefuseABadPort(t *testing.T) {
	for _, args := range [][]string{{"share", "http"}, {"unshare", "70000"}} {
		_, err := clitest.Run(PortCmd, "", args...)
		assert.ErrorContains(t, err, "invalid port", args[0])
	}

	_, err := clitest.Run(PortCmd, "", "share")
	assert.ErrorContains(t, err, "accepts 1 arg(s)")
}

func TestUnsharePrintsThePortPrivate(t *testing.T) {
	testutil.ServeSession(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, `{"ok":true}`) }))

	out, err := clitest.Run(PortCmd, "", "unshare", "8000")

	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(out, "Port 8000 is private"), out)
}

func TestRenderListShowsVisibilityURLsAndTabState(t *testing.T) {
	running := false
	listing := &port.Listing{Enabled: true, LoginRequired: true, Shares: []port.Share{
		{Port: 8000, Access: "public", Source: port.Source{Kind: "cli", Pid: 9}, URLs: []string{"https://8000.ws.example"}},
		{Port: 9000, Access: "public", Source: port.Source{Kind: "editor"}, URLs: []string{"https://9000.ws.example"}},
	}}
	tab := []port.TabPort{{Port: 8000, Privacy: "public", Forwarded: true, Running: &running}, {Port: 3000, Privacy: "private"}}

	var out strings.Builder
	renderList(&out, []port.Listener{{Port: 3000, Addresses: []string{"::1"}}, {Port: 8000, Addresses: []string{"0.0.0.0"}}}, listing, tab, nil)
	text := testutil.StripAnsi(out.String())

	for _, want := range []string{"public (ws-cli pid 9)", "https://8000.ws.example", "forwarded, public, unused", "detected", "public (the editor)", "not listening", "private"} {
		assert.Assert(t, strings.Contains(text, want), "missing %q in\n%s", want, text)
	}
}

func TestRenderListSaysWhenSharingIsOffOrTheTabIsUnknown(t *testing.T) {
	var out strings.Builder
	renderList(&out, []port.Listener{{Port: 3000, Addresses: []string{"::1"}}}, &port.Listing{}, nil, io.ErrUnexpectedEOF)
	text := testutil.StripAnsi(out.String())

	assert.Assert(t, strings.Contains(text, "WS_SERVER_PROXY_ALLOW_PUBLIC"), text)
	assert.Assert(t, strings.Contains(text, "unknown"), text)
}

func TestClearReportsTheClosedPorts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	for body, want := range map[string]string{`{"closed":[3000,8000]}`: "3000, 8000", `{"closed":[]}`: "No unused ports"} {
		testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, body) }))

		out, err := clitest.Run(PortCmd, "", "clear")

		assert.NilError(t, err)
		assert.Assert(t, strings.Contains(out, want), out)
	}
}
