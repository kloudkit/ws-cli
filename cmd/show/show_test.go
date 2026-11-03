package show

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/testutil"
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
      token_file:
        type: path
        default: "~/.ws/secrets/token"
        description: Path to the token file.
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

const (
	deprecatedLine = "Deprecated: [WS_PORT] use [WS_SERVER_PORT] instead\n"
	bothSetLine    = "Both [WS_PORT] (deprecated) and [WS_SERVER_PORT] are set. Aborting\n"
	charsetPackage = `[a-zA-Z0-9][a-zA-Z0-9._+-]*`
	charsetDomain  = `[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?`
	charsetIdent   = `[a-zA-Z0-9_-]+`
)

func installEnvFixture(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "env.reference.yaml")
	assert.NilError(t, os.WriteFile(path, []byte(envFixture), 0o644))
	t.Setenv("WS__INTERNAL_ENV_REFERENCE", path)
}

func runShow(t *testing.T, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	original := osExit
	osExit = func(code int) { exit = code }
	t.Cleanup(func() { osExit = original })

	config.ResetWarnedAliases()
	originalWriter := config.SetDeprecationWriter(os.Stderr)
	t.Cleanup(func() { config.SetDeprecationWriter(originalWriter) })

	testutil.ResetCommandFlags(ShowCmd)
	ShowCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) { f.Changed = false; _ = f.Value.Set(f.DefValue) })

	var outBuf, errBuf bytes.Buffer
	ShowCmd.SetOut(&outBuf)
	ShowCmd.SetErr(&errBuf)
	ShowCmd.SetArgs(args)
	_ = ShowCmd.Execute()

	return testutil.StripAnsi(outBuf.String()), testutil.StripAnsi(errBuf.String()), exit
}

func skip(key string) string {
	return "Skipped: env [" + key + "] not set"
}

func ptr(s string) *string { return &s }

type showCase struct {
	name     string
	env      []string
	setup    func(t *testing.T) []string
	args     []string
	exit     int
	failed   bool
	out      *string
	stderr   *string
	has      []string
	lacks    []string
	errHas   []string
	errLacks []string
}

func (c showCase) run(t *testing.T) {
	installEnvFixture(t)

	env := c.env
	if c.setup != nil {
		env = append(env, c.setup(t)...)
	}

	for _, kv := range env {
		key, value, _ := strings.Cut(kv, "=")
		t.Setenv(key, value)
	}

	stdout, stderr, exit := runShow(t, c.args...)

	if c.failed {
		assert.Assert(t, exit != 0 || stderr != "", "want failure, got exit=%d stderr=%q", exit, stderr)
	} else {
		assert.Equal(t, exit, c.exit, "stderr=%q", stderr)
	}

	if c.out != nil {
		assert.Equal(t, strings.TrimSpace(stdout), *c.out)
	}

	if c.stderr != nil {
		assert.Equal(t, stderr, *c.stderr)
	}

	for _, want := range c.has {
		assert.Assert(t, strings.Contains(stdout, want), "want %q in %q", want, stdout)
	}

	for _, want := range c.lacks {
		assert.Assert(t, !strings.Contains(stdout, want), "want no %q in %q", want, stdout)
	}

	for _, want := range c.errHas {
		assert.Assert(t, strings.Contains(stderr, want), "want %q in stderr %q", want, stderr)
	}

	for _, want := range c.errLacks {
		assert.Assert(t, !strings.Contains(stderr, want), "want no %q in stderr %q", want, stderr)
	}
}

func passwordFile(t *testing.T) []string {
	file := filepath.Join(t.TempDir(), "pw")
	assert.NilError(t, os.WriteFile(file, []byte("super-secret\n"), 0o600))

	return []string{"WS_AUTH_PASSWORD=file:" + file}
}

func conventionSecret(t *testing.T) []string {
	root := t.TempDir()
	assert.NilError(t, os.MkdirAll(filepath.Join(root, "auth"), 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(root, "auth", "password"), []byte("conv-secret\n"), 0o600))

	return []string{"WS__INTERNAL_SECRETS_ROOT=" + root, "WS_AUTH_PASSWORD="}
}

func TestShow(t *testing.T) {
	const feats = "features.additional_features"

	cases := []showCase{
		{name: "PathHomeEnv", env: []string{"WS_SERVER_ROOT=/app"}, args: []string{"path", "home"}, has: []string{"/app"}},
		{name: "PathHomeDefault", env: []string{"WS_SERVER_ROOT="}, args: []string{"path", "home"}, has: []string{"/workspace"}},
		{name: "PathHomeRaw", env: []string{"WS_SERVER_ROOT=/app"}, args: []string{"path", "home", "--raw"}, out: ptr("/app")},
		{name: "PathVscodeHome", env: []string{"HOME=/app"}, args: []string{"path", "vscode-settings"}, has: []string{"/app/.local/share/ws-server/User/settings.json", "User"}},
		{name: "PathVscodeNoHome", env: []string{"HOME="}, args: []string{"path", "vscode-settings"}, has: []string{"/home/kloud/.local/share/ws-server/User/settings.json"}},
		{name: "PathVscodeWorkspace", args: []string{"path", "vscode-settings", "--workspace"}, has: []string{"/workspace/.vscode/settings.json", "Workspace"}},

		{name: "ValueDefault", env: []string{"WS_SERVER_ROOT="}, args: []string{"env", "server.root", "--value"}, out: ptr("/workspace")},
		{name: "ValueEnv", env: []string{"WS_SERVER_ROOT=/custom"}, args: []string{"env", "server.root", "--value"}, out: ptr("/custom")},
		{name: "ValueMultiSegment", env: []string{"WS_AUTH_GITHUB_TOKEN_FILE=/run/secrets/gh"}, args: []string{"env", "auth.github_token_file", "--value"}, out: ptr("/run/secrets/gh")},
		{name: "ValueEmptyNoSkip", env: []string{"WS_FEATURES_ADDITIONAL_FEATURES="}, args: []string{"env", feats, "--value"}, out: ptr(""), stderr: ptr("")},
		{name: "ValuePathExpanded", env: []string{"HOME=/home/kloud", "WS_SECRETS_TOKEN_FILE="}, args: []string{"env", "secrets.token_file", "--value"}, out: ptr("/home/kloud/.ws/secrets/token")},
		{name: "ValueSecretFile", setup: passwordFile, args: []string{"env", "auth.password", "--value"}, out: ptr("super-secret")},
		{name: "ValueNonSecretFilePrefix", env: []string{"WS_SERVER_ROOT=file:/tmp/x"}, args: []string{"env", "server.root", "--value"}, failed: true, errHas: []string{"file: prefix is only valid on secret properties", "WS_SERVER_ROOT"}},

		{name: "AsBoolTrue", env: []string{"WS_SERVER_ROOT=true"}, args: []string{"env", "server.root", "--as", "bool"}},
		{name: "AsBoolFalse", env: []string{"WS_SERVER_ROOT=false"}, args: []string{"env", "server.root", "--as", "bool"}, exit: 1},
		{name: "AsBoolUnsetNoSkip", env: []string{"WS_FEATURES_ADDITIONAL_FEATURES="}, args: []string{"env", feats, "--as", "bool"}, failed: true},
		{name: "AsInt", env: []string{"WS_METRICS_PORT="}, args: []string{"env", "metrics.port", "--as", "int"}, out: ptr("9100")},
		{name: "AsList", env: []string{"WS_FEATURES_ADDITIONAL_FEATURES=tshark gh helm-extras"}, args: []string{"env", feats, "--as", "list"}, out: ptr("tshark\ngh\nhelm-extras")},
		{name: "AsListVerbatim", env: []string{"WS_FEATURES_ADDITIONAL_FEATURES=a;b c"}, args: []string{"env", feats, "--as", "list"}, out: ptr("a;b\nc")},
		{name: "AsUnknown", env: []string{"WS_SERVER_ROOT=anything"}, args: []string{"env", "server.root", "--as", "garbage"}, failed: true},
		{name: "AsValueMutex", args: []string{"env", "server.root", "--value", "--as", "int"}, failed: true, errHas: []string{"none of the others can be"}},

		{name: "UnknownCheckNotConflated", env: []string{"WS_SERVER_BOGUS="}, args: []string{"env", "server.bogus", "--check"}, exit: 1, errLacks: []string{"Unknown env var"}},
		{name: "InternalKeyRejected", args: []string{"env", "WS__INTERNAL_ENV_REFERENCE"}, exit: 2, errHas: []string{"Use dotted key", "WS__INTERNAL_ENV_REFERENCE"}},
		{name: "WSKeyRejectedValue", env: []string{"WS_SERVER_PORT=9000"}, args: []string{"env", "WS_SERVER_PORT", "--value"}, exit: 2, out: ptr(""), stderr: ptr("Use dotted key [server.port] instead of [WS_SERVER_PORT]\n")},
		{name: "WSKeyRejectedPretty", args: []string{"env", "WS_SERVER_PORT"}, exit: 2, stderr: ptr("Use dotted key [server.port] instead of [WS_SERVER_PORT]\n")},

		{name: "PrettyLongDescription", env: []string{"WS_SERVER_ROOT="}, args: []string{"env", "server.root"}, has: []string{"WS_SERVER_ROOT", "Root directory for the workspace.", "/workspace", "override the default", "declared"}},
		{name: "PrettyNoLongDescription", env: []string{"WS_METRICS_PORT="}, args: []string{"env", "metrics.port"}, has: []string{"WS_METRICS_PORT", "Port on which the metrics endpoint listens.", "9100"}},
		{name: "PrettySourceEnv", env: []string{"WS_SERVER_ROOT=/from-env"}, args: []string{"env", "server.root"}, has: []string{"process", "/from-env"}},
		{name: "PrettyDeprecatedAlias", env: []string{"WS_SERVER_PORT=", "WS_PORT=9000"}, args: []string{"env", "server.port"}, has: []string{"alias"}, errHas: []string{deprecatedLine}},
		{name: "PrettyPathExpanded", env: []string{"HOME=/home/kloud", "WS_SECRETS_TOKEN_FILE="}, args: []string{"env", "secrets.token_file"}, has: []string{"/home/kloud/.ws/secrets/token", "declared"}},
		{name: "PrettyNullDefault", env: []string{"HOME=/home/kloud", "WS_AUTH_GITHUB_TOKEN_FILE="}, args: []string{"env", "auth.github_token_file"}, has: []string{"declared", "(unset)"}},
		{name: "PrettySecretFile", setup: passwordFile, args: []string{"env", "auth.password"}, has: []string{"file", "<redacted>"}, lacks: []string{"super-secret"}},
		{name: "PrettySecretConvention", setup: conventionSecret, args: []string{"env", "auth.password"}, has: []string{"mount", "<redacted>"}, lacks: []string{"conv-secret"}},

		{name: "CheckPreferred", env: []string{"WS_SERVER_PORT=9000", "WS_PORT="}, args: []string{"env", "server.port", "--check", "--deprecated", "WS_PORT"}, stderr: ptr("")},
		{name: "CheckDeprecatedOnly", env: []string{"WS_SERVER_PORT=", "WS_PORT=9000"}, args: []string{"env", "server.port", "--check", "--deprecated", "WS_PORT"}, exit: 1, stderr: ptr(deprecatedLine)},
		{name: "CheckBoth", env: []string{"WS_SERVER_PORT=9000", "WS_PORT=9001"}, args: []string{"env", "server.port", "--check", "--deprecated", "WS_PORT"}, exit: 2, stderr: ptr(bothSetLine)},
		{name: "CheckUnset", env: []string{"WS_SERVER_PORT=", "WS_PORT="}, args: []string{"env", "server.port", "--check", "--deprecated", "WS_PORT"}, exit: 1},

		{name: "OrSkipValueSet", env: []string{"WS_SERVER_ROOT=/custom"}, args: []string{"env", "server.root", "--value", "--or-skip"}, out: ptr("/custom"), stderr: ptr("")},
		{name: "OrSkipValueDefault", env: []string{"WS_SERVER_ROOT="}, args: []string{"env", "server.root", "--value", "--or-skip"}, out: ptr("/workspace")},
		{name: "OrSkipValueUnset", env: []string{"WS_FEATURES_ADDITIONAL_FEATURES="}, args: []string{"env", feats, "--value", "--or-skip"}, exit: 1, out: ptr(""), errHas: []string{skip("WS_FEATURES_ADDITIONAL_FEATURES")}},
		{name: "ValueCheckPreferred", env: []string{"WS_SERVER_PORT=9000", "WS_PORT="}, args: []string{"env", "server.port", "--value", "--check"}, out: ptr("9000"), errLacks: []string{"none of the others can be"}},
		{name: "ValueCheckAliasOnly", env: []string{"WS_SERVER_PORT=", "WS_PORT=9001"}, args: []string{"env", "server.port", "--value", "--check", "--deprecated", "WS_PORT", "--or-skip"}, out: ptr("9001"), errHas: []string{deprecatedLine}},
		{name: "ValueCheckNeither", env: []string{"WS_SERVER_PORT=", "WS_PORT="}, args: []string{"env", "server.port", "--value", "--check", "--or-skip"}, exit: 1, out: ptr(""), errHas: []string{skip("WS_SERVER_PORT")}},
		{name: "ValueCheckBoth", env: []string{"WS_SERVER_PORT=9000", "WS_PORT=9001"}, args: []string{"env", "server.port", "--value", "--check", "--deprecated", "WS_PORT", "--or-skip"}, exit: 2, stderr: ptr(bothSetLine)},

		{name: "BoolOrSkipTrue", env: []string{"WS_SERVER_ROOT=true"}, args: []string{"env", "server.root", "--as", "bool", "--or-skip"}, stderr: ptr("")},
		{name: "BoolOrSkipPaddedTrue", env: []string{"WS_SERVER_ROOT=  true  "}, args: []string{"env", "server.root", "--as", "bool", "--or-skip"}},
		{name: "BoolOrSkipFalse", env: []string{"WS_SERVER_ROOT=false"}, args: []string{"env", "server.root", "--as", "bool", "--or-skip"}, exit: 1, errLacks: []string{skip("WS_SERVER_ROOT")}},
		{name: "BoolOrSkipPaddedFalse", env: []string{"WS_SERVER_ROOT=  false  "}, args: []string{"env", "server.root", "--as", "bool", "--or-skip"}, exit: 1, errLacks: []string{skip("WS_SERVER_ROOT")}},
		{name: "BoolOrSkipWhitespace", env: []string{"WS_SERVER_ROOT=   "}, args: []string{"env", "server.root", "--as", "bool", "--or-skip"}, exit: 1, errHas: []string{skip("WS_SERVER_ROOT")}},
		{name: "BoolOrSkipUnset", env: []string{"WS_FEATURES_ADDITIONAL_FEATURES="}, args: []string{"env", feats, "--as", "bool", "--or-skip"}, exit: 1, errHas: []string{skip("WS_FEATURES_ADDITIONAL_FEATURES")}},
		{name: "BoolOrSkipNullDefault", env: []string{"WS_AUTH_GITHUB_TOKEN_FILE="}, args: []string{"env", "auth.github_token_file", "--as", "bool", "--or-skip"}, exit: 1, errHas: []string{skip("WS_AUTH_GITHUB_TOKEN_FILE")}},

		{name: "ValidateLegal", env: []string{"WS_FEATURES_ADDITIONAL_FEATURES=tshark gh helm-extras"}, args: []string{"env", feats, "--as", "list", "--validate", charsetPackage}, out: ptr("tshark\ngh\nhelm-extras"), stderr: ptr("")},
		{name: "ValidateLegalDomains", env: []string{"WS_FEATURES_ADDITIONAL_FEATURES=example.com api.example.org"}, args: []string{"env", feats, "--as", "list", "--validate", charsetDomain}, out: ptr("example.com\napi.example.org")},
		{name: "ValidateOneBadFailsClosed", env: []string{"WS_FEATURES_ADDITIONAL_FEATURES=good evil;rm alsogood"}, args: []string{"env", feats, "--as", "list", "--validate", charsetPackage}, exit: 3, out: ptr(""), errHas: []string{"Rejected: invalid item [evil;rm]"}},
	}

	for _, flags := range [][]string{{}, {"--value"}, {"--as", "bool"}, {"--as", "int"}, {"--as", "list"}, {"--value", "--check", "--or-skip"}} {
		cases = append(cases, showCase{
			name: "Unknown" + strings.Join(flags, ""), args: append([]string{"env", "server.bogus"}, flags...),
			exit: 2, out: ptr(""), stderr: ptr("Unknown env var [server.bogus]\n"),
		})
	}

	for _, leaf := range [][]string{{"path", "home"}, {"path", "vscode-settings"}, {"ip", "internal"}, {"ip", "node"}, {"env", "server.root"}} {
		cases = append(cases, showCase{name: "Raw" + strings.Join(leaf, ""), args: append(leaf, "--raw"), errLacks: []string{"unknown flag"}})
	}

	for _, bad := range []struct{ value, charset string }{
		{"bad;rm", charsetPackage},
		{"x$(touch /tmp/pwn)", charsetPackage},
		{"x`id`", charsetPackage},
		{"x$(touch${IFS}/tmp/pwn)", charsetIdent},
		{"pkg*", charsetPackage},
		{"abc\ndef", charsetIdent},
		{"-rf", charsetPackage},
		{"evil.com;drop", charsetDomain},
	} {
		cases = append(cases, showCase{
			name: "ValidateRejects" + bad.value, env: []string{"WS_FEATURES_ADDITIONAL_FEATURES=" + bad.value},
			args: []string{"env", feats, "--as", "list", "--delimiter", "|", "--validate", bad.charset},
			exit: 3, out: ptr(""), errHas: []string{"Rejected: invalid item [" + bad.value + "]"},
		})
	}

	for _, c := range cases {
		t.Run(c.name, c.run)
	}
}

func TestShowValueCmd(t *testing.T) {
	run := func(getter func() (string, error)) (string, error) {
		cmd := makeValueCmd("ip", "Display an IP address", "", "Node IP Address", "Address", getter)
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(nil)
		err := cmd.Execute()
		return testutil.StripAnsi(out.String()), err
	}

	out, err := run(func() (string, error) { return "192.168.1.5", nil })
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(out, "Node IP Address") && strings.Contains(out, "192.168.1.5"), out)

	_, err = run(func() (string, error) { return "", errors.New("simulated") })
	assert.ErrorContains(t, err, "simulated")
}
