package show

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func _runIPCmd(t *testing.T, title string, getter func() (string, error)) (stdout, stderr string, err error) {
	t.Helper()
	cmd := makeIPCmd("ip", "Display an IP address", title, getter)

	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{})
	err = cmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestShowIP_Internal_RendersValue(t *testing.T) {
	getter := func() (string, error) { return "10.0.0.1", nil }

	stdout, _, err := _runIPCmd(t, "Internal IP Address", getter)
	assert.NilError(t, err)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(strings.ToUpper(plain), "INTERNAL IP ADDRESS"), "want title, got: %q", plain)
	assert.Assert(t, strings.Contains(plain, "10.0.0.1"), "want IP value, got: %q", plain)
}

func TestShowIP_Node_RendersValue(t *testing.T) {
	getter := func() (string, error) { return "192.168.1.5", nil }

	stdout, _, err := _runIPCmd(t, "Node IP Address", getter)
	assert.NilError(t, err)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(strings.ToUpper(plain), "NODE IP ADDRESS"), "want title, got: %q", plain)
	assert.Assert(t, strings.Contains(plain, "192.168.1.5"), "want IP value, got: %q", plain)
}

func TestShowIP_GetterError_NonZeroExit(t *testing.T) {
	getter := func() (string, error) { return "", errors.New("simulated") }

	_, _, err := _runIPCmd(t, "Internal IP Address", getter)
	assert.Assert(t, err != nil, "want non-nil error (maps to non-zero exit at root)")
	assert.ErrorContains(t, err, "simulated")
}
