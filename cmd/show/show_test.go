package show

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/config"
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
        description: Root directory for the workspace.
        longDescription: |
          Accepts a **path** to override the default ` + "`/workspace`" + ` location.
      port:
        type: integer
        default: 8080
        description: Port on which the web server listens.
  metrics:
    properties:
      port:
        type: integer
        default: 9100
        description: Port on which the metrics endpoint listens.
  features:
    properties:
      additional_features:
        type: string
        default: null
        delimiter: " "
  secrets:
    properties:
      vault:
        type: path
        default: "~/.ws/vault/secrets.yaml"
        description: Path to the encrypted vault file.
  auth:
    properties:
      github_token_file:
        type: path
        default: null
        description: Path to the GitHub token file.
      password:
        type: string
        default: null
        secret: true
        description: Plaintext password for the editor.
deprecated:
  WS_PORT:
    use: WS_SERVER_PORT
`

var _ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func _stripANSI(s string) string {
	return _ansiRE.ReplaceAllString(s, "")
}

func _extractField(plain, label string) string {
	prefix := label + ":"
	for line := range strings.SplitSeq(plain, "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), prefix); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

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

	config.ResetWarnedAliases()
	originalWriter := config.SetDeprecationWriter(os.Stderr)
	t.Cleanup(func() { config.SetDeprecationWriter(originalWriter) })

	envCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		_ = f.Value.Set(f.DefValue)
	})

	ShowCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
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

func TestShowEnv_ValueDefault(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "")

	stdout, _, exit := _runShow(t, "env", "WS_SERVER_ROOT", "--value")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "/workspace", strings.TrimSpace(stdout))
}

func TestShowEnv_ValueHonorsEnv(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "/custom")

	stdout, _, exit := _runShow(t, "env", "WS_SERVER_ROOT", "--value")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "/custom", strings.TrimSpace(stdout))
}

func TestShowEnv_AsBool_TruthyExitsZero(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "true")

	_, _, exit := _runShow(t, "env", "WS_SERVER_ROOT", "--as", "bool")
	assert.Equal(t, 0, exit)
}

func TestShowEnv_AsBool_FalsyExitsOne(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "false")

	_, _, exit := _runShow(t, "env", "WS_SERVER_ROOT", "--as", "bool")
	assert.Equal(t, 1, exit)
}

func TestShowEnv_AsInt_PrintsCanonical(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_METRICS_PORT", "")

	stdout, _, exit := _runShow(t, "env", "WS_METRICS_PORT", "--as", "int")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "9100", strings.TrimSpace(stdout))
}

func TestShowEnv_AsList_WithYAMLDelimiter(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_FEATURES_ADDITIONAL_FEATURES", "tshark gh helm-extras")

	stdout, _, exit := _runShow(t, "env", "WS_FEATURES_ADDITIONAL_FEATURES", "--as", "list")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "tshark\ngh\nhelm-extras", strings.TrimSpace(stdout))
}

func TestShowEnv_AsRejectsUnknownType(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "anything")

	_, stderr, exit := _runShow(t, "env", "WS_SERVER_ROOT", "--as", "garbage")
	assert.Assert(t, exit != 0 || stderr != "")
}

func TestShowEnv_AsAndValueMutuallyExclusive(t *testing.T) {
	_installEnvFixture(t)

	_, stderr, _ := _runShow(t, "env", "WS_SERVER_ROOT", "--value", "--as", "int")
	assert.Assert(t, strings.Contains(stderr, "none of the others can be"), "want cobra mutex error, got: %q", stderr)
}

func TestShowEnv_UnknownKey_Default(t *testing.T) {
	_installEnvFixture(t)

	stdout, stderr, exit := _runShow(t, "env", "WS_NOT_DECLARED")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "", stdout)
	assert.Equal(t, "Unknown env var [WS_NOT_DECLARED]\n", stderr)
}

func TestShowEnv_UnknownKey_Value(t *testing.T) {
	_installEnvFixture(t)

	_, stderr, exit := _runShow(t, "env", "WS_NOT_DECLARED", "--value")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Unknown env var [WS_NOT_DECLARED]\n", stderr)
}

func TestShowEnv_UnknownKey_AsBool(t *testing.T) {
	_installEnvFixture(t)

	_, stderr, exit := _runShow(t, "env", "WS_NOT_DECLARED", "--as", "bool")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Unknown env var [WS_NOT_DECLARED]\n", stderr)
}

func TestShowEnv_UnknownKey_AsInt(t *testing.T) {
	_installEnvFixture(t)

	_, stderr, exit := _runShow(t, "env", "WS_NOT_DECLARED", "--as", "int")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Unknown env var [WS_NOT_DECLARED]\n", stderr)
}

func TestShowEnv_UnknownKey_AsList(t *testing.T) {
	_installEnvFixture(t)

	_, stderr, exit := _runShow(t, "env", "WS_NOT_DECLARED", "--as", "list")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Unknown env var [WS_NOT_DECLARED]\n", stderr)
}

func TestShowEnv_UnknownKey_StderrNotStdout(t *testing.T) {
	_installEnvFixture(t)

	stdout, _, _ := _runShow(t, "env", "WS_NOT_DECLARED")
	assert.Equal(t, "", stdout)
}

func TestShowEnv_UnknownKey_NotConflatedWithCheck(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_NOT_DECLARED", "")

	_, stderr, exit := _runShow(t, "env", "WS_NOT_DECLARED", "--check")
	assert.Equal(t, 1, exit)
	assert.Assert(t, !strings.Contains(stderr, "Unknown env var"))
}

func TestShowEnv_UnknownKey_ExitCode(t *testing.T) {
	_installEnvFixture(t)

	_, _, exit := _runShow(t, "env", "WS_BOGUS_KEY_42")
	assert.Equal(t, 2, exit)
}

func TestShowEnv_InternalKeyAlsoErrors(t *testing.T) {
	_installEnvFixture(t)

	_, stderr, exit := _runShow(t, "env", "WS__INTERNAL_ENV_REFERENCE")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Unknown env var [WS__INTERNAL_ENV_REFERENCE]\n", stderr)
}

func TestShowEnv_LongDescriptionRenders_Default(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "")

	stdout, _, exit := _runShow(t, "env", "WS_SERVER_ROOT")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "WS_SERVER_ROOT"), "want key in output, got: %q", plain)
	assert.Assert(t, strings.Contains(plain, "Root directory for the workspace."), "want description, got: %q", plain)
	assert.Assert(t, strings.Contains(plain, "/workspace"), "want resolved value, got: %q", plain)
	assert.Assert(t, strings.Contains(plain, "override the default"), "want longDescription text, got: %q", plain)
}

func TestShowEnv_LongDescriptionAbsent_DefaultStillRenders(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_METRICS_PORT", "")

	stdout, _, exit := _runShow(t, "env", "WS_METRICS_PORT")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "WS_METRICS_PORT"))
	assert.Assert(t, strings.Contains(plain, "Port on which the metrics endpoint listens."))
	assert.Assert(t, strings.Contains(plain, "9100"))
}

func TestShowEnv_SourceLabel_EnvSet(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "/from-env")

	stdout, _, exit := _runShow(t, "env", "WS_SERVER_ROOT")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "process"), "want env-set source label, got: %q", plain)
}

func TestShowEnv_SourceLabel_YAMLDefault(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "")

	stdout, _, exit := _runShow(t, "env", "WS_SERVER_ROOT")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "declared"), "want yaml-default source label, got: %q", plain)
}

func TestShowEnv_SourceLabel_DeprecatedAlias(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "")
	t.Setenv("WS_PORT", "9000")

	stdout, _, exit := _runShow(t, "env", "WS_SERVER_PORT")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "alias"), "want deprecated-alias label, got: %q", plain)
}

func TestShowEnv_SourceLabel_EmptyEnvFallsBackToYAML(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "")

	stdout, _, exit := _runShow(t, "env", "WS_SERVER_ROOT")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "declared"))
	assert.Assert(t, strings.Contains(plain, "/workspace"))
}

func TestShowEnv_DeprecatedAlias_StderrAndLabel(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "")
	t.Setenv("WS_PORT", "9000")

	stdout, stderr, exit := _runShow(t, "env", "WS_SERVER_PORT")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "alias"), "want source label, got: %q", plain)
	assert.Assert(t, strings.Contains(stderr, "Deprecated: [WS_PORT] use [WS_SERVER_PORT] instead"), "want deprecation warn, got: %q", stderr)
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

func TestShowEnv_TypePath_DefaultMode_RendersExpandedValue(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_SECRETS_VAULT", "")

	stdout, _, exit := _runShow(t, "env", "WS_SECRETS_VAULT")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "/home/kloud/.ws/vault/secrets.yaml"), "want expanded value, got: %q", plain)
	assert.Assert(t, strings.Contains(plain, "declared"), "want yaml-default source label, got: %q", plain)
}

func TestShowEnv_TypePath_ValueFlag_ReturnsExpanded(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_SECRETS_VAULT", "")

	stdout, _, exit := _runShow(t, "env", "WS_SECRETS_VAULT", "--value")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "/home/kloud/.ws/vault/secrets.yaml", strings.TrimSpace(stdout))
}

func TestShowEnv_TypePath_SourceLabel_NullDefault_TypePath(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_AUTH_GITHUB_TOKEN_FILE", "")

	stdout, _, exit := _runShow(t, "env", "WS_AUTH_GITHUB_TOKEN_FILE")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "declared"), "want yaml-default source label, got: %q", plain)
	assert.Equal(t, "", _extractField(plain, "Value"), "want empty Value field, got: %q", plain)
}

func TestShowEnv_Secret_FilePrefix_ValueFlag(t *testing.T) {
	_installEnvFixture(t)
	pwFile := filepath.Join(t.TempDir(), "pw")
	assert.NilError(t, os.WriteFile(pwFile, []byte("super-secret\n"), 0o600))
	t.Setenv("WS_AUTH_PASSWORD", "file:"+pwFile)

	stdout, _, exit := _runShow(t, "env", "WS_AUTH_PASSWORD", "--value")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "super-secret", strings.TrimSpace(stdout))

	t.Setenv("WS_AUTH_PASSWORD", "file:"+pwFile)
	stdout, _, exit = _runShow(t, "env", "WS_AUTH_PASSWORD")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "file"), "want env-file source label, got: %q", plain)
	assert.Assert(t, !strings.Contains(plain, "super-secret"), "want secret value redacted, got: %q", plain)
	assert.Assert(t, strings.Contains(plain, "<redacted>"), "want redaction marker, got: %q", plain)
}

func TestShowEnv_Secret_ConventionDefault_SourceLabel(t *testing.T) {
	_installEnvFixture(t)
	root := t.TempDir()
	t.Setenv("WS__INTERNAL_SECRETS_ROOT", root)
	assert.NilError(t, os.MkdirAll(filepath.Join(root, "auth"), 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(root, "auth/password"), []byte("conv-secret\n"), 0o600))
	t.Setenv("WS_AUTH_PASSWORD", "")

	stdout, _, exit := _runShow(t, "env", "WS_AUTH_PASSWORD")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "mount"), "want secret-file-default source label, got: %q", plain)
	assert.Assert(t, !strings.Contains(plain, "conv-secret"), "want secret value redacted, got: %q", plain)
	assert.Assert(t, strings.Contains(plain, "<redacted>"), "want redaction marker, got: %q", plain)
}

func TestShowEnv_NonSecret_FilePrefix_ErrorPath(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "file:/tmp/x")

	_, stderr, _ := _runShow(t, "env", "WS_SERVER_ROOT", "--value")
	assert.Assert(t, strings.Contains(stderr, "file: prefix is only valid on secret properties"),
		"want foot-gun error, got stderr=%q", stderr)
	assert.Assert(t, strings.Contains(stderr, "WS_SERVER_ROOT"), "want runtimeKey in stderr, got: %q", stderr)
}

func TestShow_RawFlagInheritedByAllLeaves(t *testing.T) {
	cases := []struct{ args []string }{
		{[]string{"path", "home", "--raw"}},
		{[]string{"path", "vscode-settings", "--raw"}},
		{[]string{"ip", "internal", "--raw"}},
		{[]string{"ip", "node", "--raw"}},
		{[]string{"env", "WS_SERVER_ROOT", "--raw"}},
	}

	for _, c := range cases {
		_installEnvFixture(t)
		_, stderr, exit := _runShow(t, c.args...)
		assert.Equal(t, 0, exit, "args=%v stderr=%q", c.args, stderr)
		assert.Assert(t, !strings.Contains(stderr, "unknown flag"), "args=%v stderr=%q", c.args, stderr)
	}
}

func TestShowEnv_CheckUnchanged(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "")
	t.Setenv("WS_PORT", "9000")

	_, stderr, exit := _runShow(t, "env", "WS_SERVER_PORT", "--check", "--deprecated", "WS_PORT")
	assert.Equal(t, 1, exit)
	assert.Equal(t, "Deprecated: [WS_PORT] use [WS_SERVER_PORT] instead\n", stderr)

	t.Setenv("WS_SERVER_PORT", "9000")
	t.Setenv("WS_PORT", "9001")

	_, stderr, exit = _runShow(t, "env", "WS_SERVER_PORT", "--check", "--deprecated", "WS_PORT")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Both [WS_PORT] (deprecated) and [WS_SERVER_PORT] are set\n. Aborting\n", stderr)
}
