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

	stdout, _, exit := _runShow(t, "env", "server.root", "--value")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "/workspace", strings.TrimSpace(stdout))
}

func TestShowEnv_ValueHonorsEnv(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "/custom")

	stdout, _, exit := _runShow(t, "env", "server.root", "--value")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "/custom", strings.TrimSpace(stdout))
}

func TestShowEnv_AsBool_TruthyExitsZero(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "true")

	_, _, exit := _runShow(t, "env", "server.root", "--as", "bool")
	assert.Equal(t, 0, exit)
}

func TestShowEnv_AsBool_FalsyExitsOne(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "false")

	_, _, exit := _runShow(t, "env", "server.root", "--as", "bool")
	assert.Equal(t, 1, exit)
}

func TestShowEnv_AsInt_PrintsCanonical(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_METRICS_PORT", "")

	stdout, _, exit := _runShow(t, "env", "metrics.port", "--as", "int")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "9100", strings.TrimSpace(stdout))
}

func TestShowEnv_AsList_WithYAMLDelimiter(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_FEATURES_ADDITIONAL_FEATURES", "tshark gh helm-extras")

	stdout, _, exit := _runShow(t, "env", "features.additional_features", "--as", "list")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "tshark\ngh\nhelm-extras", strings.TrimSpace(stdout))
}

func TestShowEnv_AsRejectsUnknownType(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "anything")

	_, stderr, exit := _runShow(t, "env", "server.root", "--as", "garbage")
	assert.Assert(t, exit != 0 || stderr != "")
}

func TestShowEnv_AsAndValueMutuallyExclusive(t *testing.T) {
	_installEnvFixture(t)

	_, stderr, _ := _runShow(t, "env", "server.root", "--value", "--as", "int")
	assert.Assert(t, strings.Contains(stderr, "none of the others can be"), "want cobra mutex error, got: %q", stderr)
}

func TestShowEnv_UnknownKey_Default(t *testing.T) {
	_installEnvFixture(t)

	stdout, stderr, exit := _runShow(t, "env", "server.bogus")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "", stdout)
	assert.Equal(t, "Unknown env var [server.bogus]\n", stderr)
}

func TestShowEnv_UnknownKey_Value(t *testing.T) {
	_installEnvFixture(t)

	_, stderr, exit := _runShow(t, "env", "server.bogus", "--value")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Unknown env var [server.bogus]\n", stderr)
}

func TestShowEnv_UnknownKey_AsBool(t *testing.T) {
	_installEnvFixture(t)

	_, stderr, exit := _runShow(t, "env", "server.bogus", "--as", "bool")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Unknown env var [server.bogus]\n", stderr)
}

func TestShowEnv_UnknownKey_AsInt(t *testing.T) {
	_installEnvFixture(t)

	_, stderr, exit := _runShow(t, "env", "server.bogus", "--as", "int")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Unknown env var [server.bogus]\n", stderr)
}

func TestShowEnv_UnknownKey_AsList(t *testing.T) {
	_installEnvFixture(t)

	_, stderr, exit := _runShow(t, "env", "server.bogus", "--as", "list")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Unknown env var [server.bogus]\n", stderr)
}

func TestShowEnv_UnknownKey_StderrNotStdout(t *testing.T) {
	_installEnvFixture(t)

	stdout, _, _ := _runShow(t, "env", "server.bogus")
	assert.Equal(t, "", stdout)
}

func TestShowEnv_UnknownKey_NotConflatedWithCheck(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_BOGUS", "")

	_, stderr, exit := _runShow(t, "env", "server.bogus", "--check")
	assert.Equal(t, 1, exit)
	assert.Assert(t, !strings.Contains(stderr, "Unknown env var"))
}

func TestShowEnv_UnknownKey_ExitCode(t *testing.T) {
	_installEnvFixture(t)

	_, _, exit := _runShow(t, "env", "server.bogus")
	assert.Equal(t, 2, exit)
}

func TestShowEnv_InternalKeyRejectedAsWSQuery(t *testing.T) {
	_installEnvFixture(t)

	_, stderr, exit := _runShow(t, "env", "WS__INTERNAL_ENV_REFERENCE")
	assert.Equal(t, 2, exit)
	assert.Assert(t, strings.Contains(stderr, "Use dotted key"), "want WS_*-reject hint, got: %q", stderr)
	assert.Assert(t, strings.Contains(stderr, "WS__INTERNAL_ENV_REFERENCE"), "want echoed input, got: %q", stderr)
}

func TestShowEnv_WSQueryKey_Rejected_Value(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "9000")

	stdout, stderr, exit := _runShow(t, "env", "WS_SERVER_PORT", "--value")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "", strings.TrimSpace(stdout))
	assert.Equal(t, "Use dotted key [server.port] instead of [WS_SERVER_PORT]\n", stderr)
}

func TestShowEnv_WSQueryKey_Rejected_Pretty(t *testing.T) {
	_installEnvFixture(t)

	_, stderr, exit := _runShow(t, "env", "WS_SERVER_PORT")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Use dotted key [server.port] instead of [WS_SERVER_PORT]\n", stderr)
}

func TestShowEnv_Dotted_ResolverReadsWSEnvVar(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "7777")

	stdout, _, exit := _runShow(t, "env", "server.port", "--value")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "7777", strings.TrimSpace(stdout))
}

func TestShowEnv_Dotted_MultiSegmentProp(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_AUTH_GITHUB_TOKEN_FILE", "/run/secrets/gh")

	stdout, _, exit := _runShow(t, "env", "auth.github_token_file", "--value")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "/run/secrets/gh", strings.TrimSpace(stdout))
}

func TestShowEnv_Dotted_CheckDeprecated_StaysRawWS(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "")
	t.Setenv("WS_PORT", "9000")

	_, stderr, exit := _runShow(t, "env", "server.port", "--check", "--deprecated", "WS_PORT")
	assert.Equal(t, 1, exit)
	assert.Equal(t, "Deprecated: [WS_PORT] use [WS_SERVER_PORT] instead\n", stderr)
}

func TestShowEnv_LongDescriptionRenders_Default(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "")

	stdout, _, exit := _runShow(t, "env", "server.root")
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

	stdout, _, exit := _runShow(t, "env", "metrics.port")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "WS_METRICS_PORT"))
	assert.Assert(t, strings.Contains(plain, "Port on which the metrics endpoint listens."))
	assert.Assert(t, strings.Contains(plain, "9100"))
}

func TestShowEnv_SourceLabel_EnvSet(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "/from-env")

	stdout, _, exit := _runShow(t, "env", "server.root")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "process"), "want env-set source label, got: %q", plain)
}

func TestShowEnv_SourceLabel_YAMLDefault(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "")

	stdout, _, exit := _runShow(t, "env", "server.root")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "declared"), "want yaml-default source label, got: %q", plain)
}

func TestShowEnv_SourceLabel_DeprecatedAlias(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "")
	t.Setenv("WS_PORT", "9000")

	stdout, _, exit := _runShow(t, "env", "server.port")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "alias"), "want deprecated-alias label, got: %q", plain)
}

func TestShowEnv_SourceLabel_EmptyEnvFallsBackToYAML(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "")

	stdout, _, exit := _runShow(t, "env", "server.root")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "declared"))
	assert.Assert(t, strings.Contains(plain, "/workspace"))
}

func TestShowEnv_DeprecatedAlias_StderrAndLabel(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "")
	t.Setenv("WS_PORT", "9000")

	stdout, stderr, exit := _runShow(t, "env", "server.port")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "alias"), "want source label, got: %q", plain)
	assert.Assert(t, strings.Contains(stderr, "Deprecated: [WS_PORT] use [WS_SERVER_PORT] instead"), "want deprecation warn, got: %q", stderr)
}

func TestShowEnv_CheckPreferredSet(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "9000")
	t.Setenv("WS_PORT", "")

	_, stderr, exit := _runShow(t, "env", "server.port", "--check", "--deprecated", "WS_PORT")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "", stderr)
}

func TestShowEnv_CheckDeprecatedOnly(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "")
	t.Setenv("WS_PORT", "9000")

	_, stderr, exit := _runShow(t, "env", "server.port", "--check", "--deprecated", "WS_PORT")
	assert.Equal(t, 1, exit)
	assert.Equal(t, "Deprecated: [WS_PORT] use [WS_SERVER_PORT] instead\n", stderr)
}

func TestShowEnv_CheckBothSet(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "9000")
	t.Setenv("WS_PORT", "9001")

	_, stderr, exit := _runShow(t, "env", "server.port", "--check", "--deprecated", "WS_PORT")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Both [WS_PORT] (deprecated) and [WS_SERVER_PORT] are set\n. Aborting\n", stderr)
}

func TestShowEnv_CheckUnset(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "")
	t.Setenv("WS_PORT", "")

	_, _, exit := _runShow(t, "env", "server.port", "--check", "--deprecated", "WS_PORT")
	assert.Equal(t, 1, exit)
}

func TestShowEnv_TypePath_DefaultMode_RendersExpandedValue(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_SECRETS_VAULT", "")

	stdout, _, exit := _runShow(t, "env", "secrets.vault")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "/home/kloud/.ws/vault/secrets.yaml"), "want expanded value, got: %q", plain)
	assert.Assert(t, strings.Contains(plain, "declared"), "want yaml-default source label, got: %q", plain)
}

func TestShowEnv_TypePath_ValueFlag_ReturnsExpanded(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_SECRETS_VAULT", "")

	stdout, _, exit := _runShow(t, "env", "secrets.vault", "--value")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "/home/kloud/.ws/vault/secrets.yaml", strings.TrimSpace(stdout))
}

func TestShowEnv_TypePath_SourceLabel_NullDefault_TypePath(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_AUTH_GITHUB_TOKEN_FILE", "")

	stdout, _, exit := _runShow(t, "env", "auth.github_token_file")
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

	stdout, _, exit := _runShow(t, "env", "auth.password", "--value")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "super-secret", strings.TrimSpace(stdout))

	t.Setenv("WS_AUTH_PASSWORD", "file:"+pwFile)
	stdout, _, exit = _runShow(t, "env", "auth.password")
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

	stdout, _, exit := _runShow(t, "env", "auth.password")
	assert.Equal(t, 0, exit)
	plain := _stripANSI(stdout)
	assert.Assert(t, strings.Contains(plain, "mount"), "want secret-file-default source label, got: %q", plain)
	assert.Assert(t, !strings.Contains(plain, "conv-secret"), "want secret value redacted, got: %q", plain)
	assert.Assert(t, strings.Contains(plain, "<redacted>"), "want redaction marker, got: %q", plain)
}

func TestShowEnv_NonSecret_FilePrefix_ErrorPath(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "file:/tmp/x")

	_, stderr, _ := _runShow(t, "env", "server.root", "--value")
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
		{[]string{"env", "server.root", "--raw"}},
	}

	for _, c := range cases {
		_installEnvFixture(t)
		_, stderr, exit := _runShow(t, c.args...)
		assert.Equal(t, 0, exit, "args=%v stderr=%q", c.args, stderr)
		assert.Assert(t, !strings.Contains(stderr, "unknown flag"), "args=%v stderr=%q", c.args, stderr)
	}
}

func _hasSkip(stderr, key string) bool {
	return strings.Contains(_stripANSI(stderr), "Skipped: env ["+key+"] not set")
}

func TestShowEnv_ValueOrSkip_ValueSet_PrintsExitsZero(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "/custom")

	stdout, stderr, exit := _runShow(t, "env", "server.root", "--value", "--or-skip")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "/custom", strings.TrimSpace(stdout))
	assert.Equal(t, "", stderr)
}

func TestShowEnv_ValueOrSkip_DefaultPresent_PrintsExitsZero(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "")

	stdout, _, exit := _runShow(t, "env", "server.root", "--value", "--or-skip")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "/workspace", strings.TrimSpace(stdout))
}

func TestShowEnv_ValueOrSkip_Unset_Skips(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_FEATURES_ADDITIONAL_FEATURES", "")

	stdout, stderr, exit := _runShow(t, "env", "features.additional_features", "--value", "--or-skip")
	assert.Equal(t, 1, exit)
	assert.Equal(t, "", strings.TrimSpace(stdout))
	assert.Assert(t, _hasSkip(stderr, "WS_FEATURES_ADDITIONAL_FEATURES"), "want skip breadcrumb, got: %q", stderr)
}

func TestShowEnv_ValueCheckOrSkip_PreferredSet_ExitsZeroPrints(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "9000")
	t.Setenv("WS_PORT", "")

	stdout, _, exit := _runShow(t, "env", "server.port", "--value", "--check")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "9000", strings.TrimSpace(stdout))
}

func TestShowEnv_ValueCheckOrSkip_DeprecatedAliasOnly_ExitsZeroPrints(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "")
	t.Setenv("WS_PORT", "9001")

	stdout, stderr, exit := _runShow(t, "env", "server.port", "--value", "--check", "--deprecated", "WS_PORT", "--or-skip")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "9001", strings.TrimSpace(stdout))
	assert.Assert(t, strings.Contains(stderr, "Deprecated: [WS_PORT] use [WS_SERVER_PORT] instead"), "want deprecation line, got: %q", stderr)
}

func TestShowEnv_ValueCheckOrSkip_NeitherSet_Skips(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "")
	t.Setenv("WS_PORT", "")

	stdout, stderr, exit := _runShow(t, "env", "server.port", "--value", "--check", "--or-skip")
	assert.Equal(t, 1, exit)
	assert.Equal(t, "", strings.TrimSpace(stdout))
	assert.Assert(t, _hasSkip(stderr, "WS_SERVER_PORT"), "want skip breadcrumb, got: %q", stderr)
}

func TestShowEnv_ValueCheckOrSkip_BothSet_ExitsTwo(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "9000")
	t.Setenv("WS_PORT", "9001")

	_, stderr, exit := _runShow(t, "env", "server.port", "--value", "--check", "--deprecated", "WS_PORT", "--or-skip")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Both [WS_PORT] (deprecated) and [WS_SERVER_PORT] are set\n. Aborting\n", stderr)
}

func TestShowEnv_ValueCheckOrSkip_UnknownKey_ExitsTwo(t *testing.T) {
	_installEnvFixture(t)

	_, stderr, exit := _runShow(t, "env", "server.bogus", "--value", "--check", "--or-skip")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Unknown env var [server.bogus]\n", stderr)
}

func TestShowEnv_ValueAndCheckNowCompatible(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "9000")

	_, stderr, exit := _runShow(t, "env", "server.port", "--value", "--check")
	assert.Equal(t, 0, exit)
	assert.Assert(t, !strings.Contains(stderr, "none of the others can be"), "want no mutex error, got: %q", stderr)
}

func TestShowEnv_AsBoolOrSkip_Truthy_ExitsZero(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "true")

	_, stderr, exit := _runShow(t, "env", "server.root", "--as", "bool", "--or-skip")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "", stderr)
}

func TestShowEnv_AsBoolOrSkip_Falsy_ExitsOne_NotSkip(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "false")

	_, stderr, exit := _runShow(t, "env", "server.root", "--as", "bool", "--or-skip")
	assert.Equal(t, 1, exit)
	assert.Assert(t, !_hasSkip(stderr, "WS_SERVER_ROOT"), "set-to-false must be silent (not a skip), got: %q", stderr)
}

func TestShowEnv_AsBoolOrSkip_Unset_Skips(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_FEATURES_ADDITIONAL_FEATURES", "")

	_, stderr, exit := _runShow(t, "env", "features.additional_features", "--as", "bool", "--or-skip")
	assert.Equal(t, 1, exit)
	assert.Assert(t, _hasSkip(stderr, "WS_FEATURES_ADDITIONAL_FEATURES"), "want skip breadcrumb, got: %q", stderr)
}

func TestShowEnv_AsBoolOrSkip_OperatorOnlyDefault_Skips(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_AUTH_GITHUB_TOKEN_FILE", "")

	_, stderr, exit := _runShow(t, "env", "auth.github_token_file", "--as", "bool", "--or-skip")
	assert.Equal(t, 1, exit)
	assert.Assert(t, _hasSkip(stderr, "WS_AUTH_GITHUB_TOKEN_FILE"), "want skip breadcrumb, got: %q", stderr)
}

func TestShowEnv_AsBoolOrSkip_PaddedTruthy_Trims(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "  true  ")

	_, _, exit := _runShow(t, "env", "server.root", "--as", "bool", "--or-skip")
	assert.Equal(t, 0, exit)
}

func TestShowEnv_AsBoolOrSkip_PaddedFalsy_Trims(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "  false  ")

	_, stderr, exit := _runShow(t, "env", "server.root", "--as", "bool", "--or-skip")
	assert.Equal(t, 1, exit)
	assert.Assert(t, !_hasSkip(stderr, "WS_SERVER_ROOT"), "padded-false is definite-false, not a skip, got: %q", stderr)
}

func TestShowEnv_AsBoolOrSkip_WhitespaceOnly_TreatedAsUnset(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_ROOT", "   ")

	_, stderr, exit := _runShow(t, "env", "server.root", "--as", "bool", "--or-skip")
	assert.Equal(t, 1, exit)
	assert.Assert(t, _hasSkip(stderr, "WS_SERVER_ROOT"), "whitespace-only resolves as unset → skip, got: %q", stderr)
}

func TestShowEnv_AsBool_Unset_NoOrSkip_StillErrors(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_FEATURES_ADDITIONAL_FEATURES", "")

	_, stderr, exit := _runShow(t, "env", "features.additional_features", "--as", "bool")
	assert.Assert(t, exit != 0 || stderr != "", "without --or-skip, unset bool must error/non-zero, got exit=%d stderr=%q", exit, stderr)
}

func TestShowEnv_Value_NoOrSkip_EmptyStillPrintsExitsZero(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_FEATURES_ADDITIONAL_FEATURES", "")

	stdout, stderr, exit := _runShow(t, "env", "features.additional_features", "--value")
	assert.Equal(t, 0, exit)
	assert.Equal(t, "", strings.TrimSpace(stdout))
	assert.Equal(t, "", stderr)
}

const (
	_charsetPackage    = `[a-zA-Z0-9][a-zA-Z0-9._+-]*`
	_charsetDomain     = `[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?`
	_charsetIdentifier = `[a-zA-Z0-9_-]+`
)

func _runValidate(t *testing.T, value, charset string) (stdout, stderr string, exit int) {
	t.Helper()
	_installEnvFixture(t)
	t.Setenv("WS_FEATURES_ADDITIONAL_FEATURES", value)
	return _runShow(t, "env", "features.additional_features", "--as", "list", "--delimiter", "|", "--validate", charset)
}

func _assertRejected(t *testing.T, stdout, stderr string, exit int, token string) {
	t.Helper()
	assert.Assert(t, exit != 0, "want non-zero reject exit, got 0; stdout=%q", stdout)
	assert.Equal(t, "", strings.TrimSpace(stdout), "fail-closed: no tokens emitted, got: %q", stdout)
	assert.Assert(t, strings.Contains(stderr, "Rejected: invalid item ["+token+"]"), "want rejection line for %q, got: %q", token, stderr)
}

func TestShowEnv_Validate_Semicolon_Rejected(t *testing.T) {
	stdout, stderr, exit := _runValidate(t, "bad;rm", _charsetPackage)
	_assertRejected(t, stdout, stderr, exit, "bad;rm")
}

func TestShowEnv_Validate_CommandSub_Rejected(t *testing.T) {
	stdout, stderr, exit := _runValidate(t, "x$(touch /tmp/pwn)", _charsetPackage)
	_assertRejected(t, stdout, stderr, exit, "x$(touch /tmp/pwn)")
}

func TestShowEnv_Validate_Backtick_Rejected(t *testing.T) {
	stdout, stderr, exit := _runValidate(t, "x`id`", _charsetPackage)
	_assertRejected(t, stdout, stderr, exit, "x`id`")
}

func TestShowEnv_Validate_IFS_Rejected(t *testing.T) {
	stdout, stderr, exit := _runValidate(t, "x$(touch${IFS}/tmp/pwn)", _charsetIdentifier)
	_assertRejected(t, stdout, stderr, exit, "x$(touch${IFS}/tmp/pwn)")
}

func TestShowEnv_Validate_Glob_Rejected(t *testing.T) {
	stdout, stderr, exit := _runValidate(t, "pkg*", _charsetPackage)
	_assertRejected(t, stdout, stderr, exit, "pkg*")
}

func TestShowEnv_Validate_Newline_Rejected(t *testing.T) {
	stdout, stderr, exit := _runValidate(t, "abc\ndef", _charsetIdentifier)
	_assertRejected(t, stdout, stderr, exit, "abc\ndef")
}

func TestShowEnv_Validate_LeadingDash_Rejected(t *testing.T) {
	stdout, stderr, exit := _runValidate(t, "-rf", _charsetPackage)
	_assertRejected(t, stdout, stderr, exit, "-rf")
}

func TestShowEnv_Validate_DomainCharset_RejectsMetachar(t *testing.T) {
	stdout, stderr, exit := _runValidate(t, "evil.com;drop", _charsetDomain)
	_assertRejected(t, stdout, stderr, exit, "evil.com;drop")
}

func TestShowEnv_Validate_LegalTokens_PassThrough(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_FEATURES_ADDITIONAL_FEATURES", "tshark gh helm-extras")

	stdout, stderr, exit := _runShow(t, "env", "features.additional_features", "--as", "list", "--validate", _charsetPackage)
	assert.Equal(t, 0, exit)
	assert.Equal(t, "tshark\ngh\nhelm-extras", strings.TrimSpace(stdout))
	assert.Equal(t, "", stderr)
}

func TestShowEnv_Validate_LegalDomains_PassThrough(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_FEATURES_ADDITIONAL_FEATURES", "example.com api.example.org")

	stdout, _, exit := _runShow(t, "env", "features.additional_features", "--as", "list", "--validate", _charsetDomain)
	assert.Equal(t, 0, exit)
	assert.Equal(t, "example.com\napi.example.org", strings.TrimSpace(stdout))
}

func TestShowEnv_Validate_OneBadToken_FailsWholeListClosed(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_FEATURES_ADDITIONAL_FEATURES", "good evil;rm alsogood")

	stdout, stderr, exit := _runShow(t, "env", "features.additional_features", "--as", "list", "--validate", _charsetPackage)
	assert.Assert(t, exit != 0)
	assert.Equal(t, "", strings.TrimSpace(stdout), "fail-closed: no partial emission, got: %q", stdout)
	assert.Assert(t, strings.Contains(stderr, "Rejected: invalid item [evil;rm]"), "want rejection of offending token, got: %q", stderr)
}

func TestShowEnv_AsList_NoValidate_Unchanged(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_FEATURES_ADDITIONAL_FEATURES", "a;b c")

	stdout, _, exit := _runShow(t, "env", "features.additional_features", "--as", "list")
	assert.Equal(t, 0, exit)
	// default space delimiter, no validation → tokens emitted verbatim
	assert.Equal(t, "a;b\nc", strings.TrimSpace(stdout))
}

func TestShowEnv_CheckUnchanged(t *testing.T) {
	_installEnvFixture(t)
	t.Setenv("WS_SERVER_PORT", "")
	t.Setenv("WS_PORT", "9000")

	_, stderr, exit := _runShow(t, "env", "server.port", "--check", "--deprecated", "WS_PORT")
	assert.Equal(t, 1, exit)
	assert.Equal(t, "Deprecated: [WS_PORT] use [WS_SERVER_PORT] instead\n", stderr)

	t.Setenv("WS_SERVER_PORT", "9000")
	t.Setenv("WS_PORT", "9001")

	_, stderr, exit = _runShow(t, "env", "server.port", "--check", "--deprecated", "WS_PORT")
	assert.Equal(t, 2, exit)
	assert.Equal(t, "Both [WS_PORT] (deprecated) and [WS_SERVER_PORT] are set\n. Aborting\n", stderr)
}
