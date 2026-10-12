package serve

import (
	"bytes"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
)

func TestServeCurrentTakenPort(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)
	defer taken.Close()

	port := strconv.Itoa(taken.Addr().(*net.TCPAddr).Port)

	testutil.ResetCommandFlags(ServeCmd)

	var out bytes.Buffer
	ServeCmd.SetOut(&out)
	ServeCmd.SetErr(&out)
	ServeCmd.SetArgs([]string{"current", "--bind", "127.0.0.1", "--port", port})

	err = ServeCmd.Execute()

	assert.ErrorContains(t, err, "address already in use")
	assert.Assert(t, !strings.Contains(out.String(), "Serving"))
}

func TestServeMetricsTakenPort(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)
	defer taken.Close()

	port := strconv.Itoa(taken.Addr().(*net.TCPAddr).Port)

	testutil.ResetCommandFlags(ServeCmd)

	var out bytes.Buffer
	ServeCmd.SetOut(&out)
	ServeCmd.SetErr(&out)
	ServeCmd.SetArgs([]string{"metrics", "--bind", "127.0.0.1", "--port", port, "--collectors", "workspace"})

	err = ServeCmd.Execute()

	assert.ErrorContains(t, err, "address already in use")
	assert.Assert(t, !strings.Contains(out.String(), "Serving"))
}
