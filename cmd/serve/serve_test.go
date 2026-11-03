package serve

import (
	"bytes"
	"net"
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
