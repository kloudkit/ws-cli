package server

import (
	"bytes"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
)

func TestServeTakenPortPrintsNothing(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)
	defer taken.Close()

	port := taken.Addr().(*net.TCPAddr).Port

	var out bytes.Buffer
	announced := false
	err = Serve(Config{Port: port, Bind: "127.0.0.1"}, http.NotFoundHandler(), &out, func(string) { announced = true })

	assert.ErrorContains(t, err, "address already in use")
	assert.Assert(t, !announced)
	assert.Equal(t, out.String(), "")
}

func TestServePrintsURLAfterBinding(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)

	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)

	var out bytes.Buffer
	urls := make(chan string, 1)
	go serve(listener, http.NotFoundHandler(), &out, func(url string) { urls <- url })

	assert.Equal(t, <-urls, "http://localhost:"+port)

	resp, err := http.Get("http://127.0.0.1:" + port)
	assert.NilError(t, err)
	resp.Body.Close()
	listener.Close()

	assert.Assert(t, strings.HasPrefix(testutil.StripAnsi(out.String()), "\n  Press Ctrl+C to stop\n"))
}

func TestDisplayURL(t *testing.T) {
	tests := []struct {
		ip   string
		want string
	}{
		{"0.0.0.0", "http://localhost:8080"},
		{"::", "http://localhost:8080"},
		{"127.0.0.1", "http://localhost:8080"},
		{"10.1.2.3", "http://10.1.2.3:8080"},
	}

	for _, tt := range tests {
		got := displayURL(&net.TCPAddr{IP: net.ParseIP(tt.ip), Port: 8080})
		assert.Equal(t, got, tt.want)
	}
}
