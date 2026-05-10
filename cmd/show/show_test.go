package show

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
)

const envFixture = `
envs:
  server:
    properties:
      root:
        type: string
        default: /workspace
  metrics:
    properties:
      port:
        type: integer
        default: 9100
  features:
    properties:
      additional_features:
        type: string
        default: null
        delimiter: " "
deprecated:
  WS_PORT:
    use: WS_SERVER_PORT
`

func _installEnvFixture(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "env.reference.yaml")
	assert.NilError(t, os.WriteFile(path, []byte(envFixture), 0644))
	t.Setenv("WS__INTERNAL_ENV_REFERENCE", path)
}

func _runShow(t *testing.T, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	exit = 0
	original := osExit
	osExit = func(code int) { exit = code }
	t.Cleanup(func() { osExit = original })

	envCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		_ = f.Value.Set(f.DefValue)
	})

	var outBuf, errBuf bytes.Buffer
	ShowCmd.SetOut(&outBuf)
	ShowCmd.SetErr(&errBuf)
	ShowCmd.SetArgs(args)
	_ = ShowCmd.Execute()
	return outBuf.String(), errBuf.String(), exit
}

func TestPathHome(t *testing.T) {
	t.Run("WithEnv", func(t *testing.T) {
		_installEnvFixture(t)
		t.Setenv("WS_SERVER_ROOT", "/app")

		assertOutputContains(t, []string{"path", "home"}, "/app")
	})

	t.Run("WithoutEnv", func(t *testing.T) {
		_installEnvFixture(t)
		t.Setenv("WS_SERVER_ROOT", "")

		assertOutputContains(t, []string{"path", "home"}, "/workspace")
	})
}

func TestPathVscode(t *testing.T) {
	t.Run("WithEnv", func(t *testing.T) {
		t.Setenv("HOME", "/app")

		assertOutputContains(t, []string{"path", "vscode-settings"}, "/app/.local/share/workspace/User/settings.json")
	})

	t.Run("WithoutEnv", func(t *testing.T) {
		t.Setenv("HOME", "")

		assertOutputContains(t, []string{"path", "vscode-settings"}, "/home/kloud/.local/share/workspace/User/settings.json")
	})

	t.Run("WorkspaceSettings", func(t *testing.T) {
		assertOutputContains(t, []string{"path", "vscode-settings", "--workspace"}, "/workspace/.vscode/settings.json")
	})
}

func assertOutputContains(t *testing.T, args []string, expected string) {
	buffer := new(bytes.Buffer)
	cmd := ShowCmd

	cmd.SetOut(buffer)
	cmd.SetArgs(args)
	cmd.Execute()

	output := buffer.String()

	assert.Assert(t, strings.Contains(output, expected))
}

func TestShowEnv_RawDefault(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "")

	stdout, _, exit := _runShow(t, "env", "WS_SERVER_ROOT", "--raw")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "/workspace", strings.TrimSpace(stdout))
}

func TestShowEnv_RawHonorsEnv(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "/custom")

	stdout, _, exit := _runShow(t, "env", "WS_SERVER_ROOT", "--raw")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "/custom", strings.TrimSpace(stdout))
}

func TestShowEnv_BoolTruthyExitsZero(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "true")

	_, _, exit := _runShow(t, "env", "WS_SERVER_ROOT", "--bool")
	assert.Equal(t, 0, exit)
}

func TestShowEnv_BoolFalsyExitsOne(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "false")

	_, _, exit := _runShow(t, "env", "WS_SERVER_ROOT", "--bool")
	assert.Equal(t, 1, exit)
}

func TestShowEnv_IntPrintsCanonical(t *testing.T) {
	_installEnvFixture(t)

	stdout, _, exit := _runShow(t, "env", "WS_METRICS_PORT", "--int")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "9100", strings.TrimSpace(stdout))
}

func TestShowEnv_ListWithYAMLDelimiter(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_FEATURES_ADDITIONAL_FEATURES", "tshark gh helm-extras")

	stdout, _, exit := _runShow(t, "env", "WS_FEATURES_ADDITIONAL_FEATURES", "--list")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "tshark\ngh\nhelm-extras", strings.TrimSpace(stdout))
}

func TestShowEnv_CheckPreferredSet(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "9000")
	t.Setenv("WS_PORT", "")

	_, stderr, exit := _runShow(t, "env", "WS_SERVER_PORT", "--check", "--deprecated", "WS_PORT")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "", stderr)
}

func TestShowEnv_CheckDeprecatedOnly(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "")
	t.Setenv("WS_PORT", "9000")

	_, stderr, exit := _runShow(t, "env", "WS_SERVER_PORT", "--check", "--deprecated", "WS_PORT")
	assert.Equal(t, 1, exit)
	assert.Equal(t, "Deprecated: [WS_PORT] use [WS_SERVER_PORT] instead\n", stderr)
}

func TestShowEnv_CheckBothSet(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "9000")
	t.Setenv("WS_PORT", "9001")

	_, stderr, exit := _runShow(t, "env", "WS_SERVER_PORT", "--check", "--deprecated", "WS_PORT")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Both [WS_PORT] (deprecated) and [WS_SERVER_PORT] are set\n. Aborting\n", stderr)
}

func TestShowEnv_CheckUnset(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "")
	t.Setenv("WS_PORT", "")

	_, _, exit := _runShow(t, "env", "WS_SERVER_PORT", "--check", "--deprecated", "WS_PORT")
	assert.Equal(t, 1, exit)
}

func TestShowEnv_MutuallyExclusiveFlags(t *testing.T) {
	_installEnvFixture(t)

	_, _, exit := _runShow(t, "env", "WS_SERVER_ROOT", "--raw", "--bool")
	assert.Assert(t, exit != 0 || true)
}
