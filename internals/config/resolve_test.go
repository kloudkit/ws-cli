package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

const sampleYAML = `
envs:
  metrics:
    properties:
      port:
        type: integer
        default: 9100
        description: Port on which the metrics endpoint listens.
        longDescription: |
          The metrics server exposes a ` + "`/`" + ` endpoint on this port.
      collectors:
        type: string
        default: null
        delimiter: ","
  server:
    properties:
      root:
        type: string
        default: /workspace
        description: Root directory for the workspace.
  features:
    properties:
      additional_features:
        type: string
        default: null
        delimiter: " "
  apt:
    properties:
      additional_packages:
        type: string
        default: null
        delimiter: " "
        description: Additional APT packages installed during startup.
        longDescription: |
          Accepts a **space-delimited** package list.
      additional_repos:
        type: string
        default: null
        delimiter: ";"
deprecated:
  WS_PORT:
    use: WS_SERVER_PORT
  WS_OLD_NOREPLACE:
    message: removed without replacement
`

func _newReference(t *testing.T) *EnvReference {
	t.Helper()
	r, err := parseEnvReference([]byte(sampleYAML))
	assert.NilError(t, err)
	return r
}

const pathYAML = `
envs:
  secrets:
    properties:
      vault:
        type: path
        default: "~/.ws/vault/secrets.yaml"
      master_key_file:
        type: path
        default: null
  server:
    properties:
      ssl_cert:
        type: path
        default: "/etc/workspace/ssl/cert.pem"
      ssl_root:
        type: path
        default: null
  features:
    properties:
      dir:
        type: path
        default: "~"
      extra_paths:
        type: path
        delimiter: ":"
        default: "~/local/bin:/opt/bin"
  paths:
    properties:
      midstring:
        type: path
        default: "/foo/~/bar"
      dollarhome:
        type: path
        default: "$HOME/.ws"
      otheruser:
        type: path
        default: "~root/x"
deprecated:
  WS_VAULT:
    use: WS_SECRETS_VAULT
`

func _installPathFixture(t *testing.T) {
	t.Helper()
	_installFixture(t, pathYAML)
}

func _installFixture(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "env.reference.yaml")
	assert.NilError(t, os.WriteFile(path, []byte(content), 0644))
	t.Setenv("WS__INTERNAL_ENV_REFERENCE", path)
}

func _captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	original := deprecationWriter
	deprecationWriter = buf
	t.Cleanup(func() { deprecationWriter = original })
	warnedAliases.Range(func(k, _ any) bool {
		warnedAliases.Delete(k)
		return true
	})
	return buf
}

func TestRuntimeKey(t *testing.T) {
	cases := []struct {
		group, prop string
		want        string
	}{
		{"metrics", "port", "WS_METRICS_PORT"},
		{"_internal", "ipc_socket", "WS__INTERNAL_IPC_SOCKET"},
		{"server", "root", "WS_SERVER_ROOT"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, RuntimeKey(c.group, c.prop))
	}
}

func TestResolve_EnvWinsOverDefault(t *testing.T) {
	r := _newReference(t)
	t.Setenv("WS_SERVER_ROOT", "/custom")
	assert.Equal(t, "/custom", r.Resolve("WS_SERVER_ROOT"))
}

func TestResolve_UnsetReturnsDefault(t *testing.T) {
	r := _newReference(t)
	assert.Equal(t, "/workspace", r.Resolve("WS_SERVER_ROOT"))
}

func TestResolve_EmptyEnvReturnsDefault(t *testing.T) {
	r := _newReference(t)
	t.Setenv("WS_SERVER_ROOT", "")
	assert.Equal(t, "/workspace", r.Resolve("WS_SERVER_ROOT"))
}

func TestResolve_NullDefaultReturnsEmpty(t *testing.T) {
	r := _newReference(t)
	t.Setenv("WS_FEATURES_ADDITIONAL_FEATURES", "")
	assert.Equal(t, "", r.Resolve("WS_FEATURES_ADDITIONAL_FEATURES"))
}

func TestResolve_UnknownKeyReturnsEmpty(t *testing.T) {
	r := _newReference(t)
	t.Setenv("WS_NOT_DECLARED", "")
	assert.Equal(t, "", r.Resolve("WS_NOT_DECLARED"))
}

func TestResolve_DeprecatedAliasUsedWithWarn(t *testing.T) {
	r := _newReference(t)
	buf := _captureWarnings(t)
	t.Setenv("WS_PORT", "8888")
	assert.Equal(t, "8888", r.Resolve("WS_SERVER_PORT"))
	assert.Equal(t, "Deprecated: [WS_PORT] use [WS_SERVER_PORT] instead\n", buf.String())
}

func TestResolve_DeprecatedWarnEmittedOnce(t *testing.T) {
	r := _newReference(t)
	buf := _captureWarnings(t)
	t.Setenv("WS_PORT", "8888")
	r.Resolve("WS_SERVER_PORT")
	r.Resolve("WS_SERVER_PORT")
	r.Resolve("WS_SERVER_PORT")
	assert.Equal(t, 1, bytes.Count(buf.Bytes(), []byte("Deprecated:")))
}

func TestResolve_BothSetPrefersPreferred(t *testing.T) {
	r := _newReference(t)
	_captureWarnings(t)
	t.Setenv("WS_SERVER_PORT", "9999")
	t.Setenv("WS_PORT", "8888")
	assert.Equal(t, "9999", r.Resolve("WS_SERVER_PORT"))
}

func TestParse_DeprecationChainCollapses(t *testing.T) {
	chained := `
envs: {}
deprecated:
  WS_A:
    use: WS_B
  WS_B:
    use: WS_C
`
	r, err := parseEnvReference([]byte(chained))
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"WS_A", "WS_B"}, _sorted(r.AliasesByPreferred["WS_C"]))
}

func TestParse_DeprecationCycleRejected(t *testing.T) {
	cyclic := `
envs: {}
deprecated:
  WS_A:
    use: WS_B
  WS_B:
    use: WS_A
`
	_, err := parseEnvReference([]byte(cyclic))
	assert.ErrorContains(t, err, "deprecation cycle")
}

func _sorted(s []string) []string {
	out := append([]string(nil), s...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func TestParse_EmptyEnvs(t *testing.T) {
	_, err := parseEnvReference([]byte("envs: {}\ndeprecated: {}\n"))
	assert.NilError(t, err)
}

func TestLoad_MissingFileReturnsError(t *testing.T) {
	t.Setenv("WS__INTERNAL_ENV_REFERENCE", filepath.Join(t.TempDir(), "missing.yaml"))
	_, err := LoadEnvReference()
	assert.ErrorContains(t, err, "cannot read")
}

func TestLoad_RespectsOverridePath(t *testing.T) {
	_installFixture(t, sampleYAML)
	r, err := LoadEnvReference()
	assert.NilError(t, err)
	assert.Equal(t, "/workspace", *r.Properties["WS_SERVER_ROOT"].Default)
}

func TestLookupProperty_KnownReturnsTrue(t *testing.T) {
	_installFixture(t, sampleYAML)
	prop, ok, err := LookupProperty("WS_SERVER_ROOT")
	assert.NilError(t, err)
	assert.Equal(t, true, ok)
	assert.Equal(t, "string", prop.Type)
}

func TestLookupProperty_UnknownReturnsFalse(t *testing.T) {
	_installFixture(t, sampleYAML)
	_, ok, err := LookupProperty("WS_NOT_DECLARED")
	assert.NilError(t, err)
	assert.Equal(t, false, ok)
}

func TestLookupProperty_InternalKeyReturnsFalse(t *testing.T) {
	_installFixture(t, sampleYAML)
	_, ok, err := LookupProperty("WS__INTERNAL_ENV_REFERENCE")
	assert.NilError(t, err)
	assert.Equal(t, false, ok)
}

func TestResolveKeyWithSource_EnvWins(t *testing.T) {
	_installFixture(t, sampleYAML)
	t.Setenv("WS_SERVER_ROOT", "/custom")
	value, source, err := ResolveKeyWithSource("WS_SERVER_ROOT")
	assert.NilError(t, err)
	assert.Equal(t, "/custom", value)
	assert.Equal(t, SourceEnv, source)
}

func TestResolveKeyWithSource_UnsetReturnsDefault(t *testing.T) {
	_installFixture(t, sampleYAML)
	t.Setenv("WS_SERVER_ROOT", "")
	value, source, err := ResolveKeyWithSource("WS_SERVER_ROOT")
	assert.NilError(t, err)
	assert.Equal(t, "/workspace", value)
	assert.Equal(t, SourceDefault, source)
}

func TestResolveKeyWithSource_EmptyEnvFallsBackToDefault(t *testing.T) {
	_installFixture(t, sampleYAML)
	t.Setenv("WS_SERVER_ROOT", "")
	value, source, err := ResolveKeyWithSource("WS_SERVER_ROOT")
	assert.NilError(t, err)
	assert.Equal(t, "/workspace", value)
	assert.Equal(t, SourceDefault, source)
}

func TestResolveKeyWithSource_DeprecatedAlias(t *testing.T) {
	_installFixture(t, sampleYAML)
	_captureWarnings(t)
	t.Setenv("WS_PORT", "8888")
	value, source, err := ResolveKeyWithSource("WS_SERVER_PORT")
	assert.NilError(t, err)
	assert.Equal(t, "8888", value)
	assert.Equal(t, SourceDeprecatedAlias, source)
}

func TestResolveKeyWithSource_NullDefaultReturnsEmpty(t *testing.T) {
	_installFixture(t, sampleYAML)
	t.Setenv("WS_FEATURES_ADDITIONAL_FEATURES", "")
	value, source, err := ResolveKeyWithSource("WS_FEATURES_ADDITIONAL_FEATURES")
	assert.NilError(t, err)
	assert.Equal(t, "", value)
	assert.Equal(t, SourceDefault, source)
}

func TestResolveSource_Label(t *testing.T) {
	assert.Equal(t, "env-set", SourceEnv.Label())
	assert.Equal(t, "deprecated-alias", SourceDeprecatedAlias.Label())
	assert.Equal(t, "yaml-default", SourceDefault.Label())
}

func TestParse_DescriptionAndLongDescriptionRoundTrip(t *testing.T) {
	r := _newReference(t)

	apt := r.Properties["WS_APT_ADDITIONAL_PACKAGES"]
	assert.Equal(t, "Additional APT packages installed during startup.", apt.Description)
	assert.Equal(t, "Accepts a **space-delimited** package list.\n", apt.LongDescription)

	metrics := r.Properties["WS_METRICS_PORT"]
	assert.Equal(t, "Port on which the metrics endpoint listens.", metrics.Description)
	assert.Equal(t, "The metrics server exposes a `/` endpoint on this port.\n", metrics.LongDescription)

	server := r.Properties["WS_SERVER_ROOT"]
	assert.Equal(t, "Root directory for the workspace.", server.Description)
	assert.Equal(t, "", server.LongDescription)

	features := r.Properties["WS_FEATURES_ADDITIONAL_FEATURES"]
	assert.Equal(t, "", features.Description)
	assert.Equal(t, "", features.LongDescription)
}

func TestParseBool(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"1", true}, {"true", true}, {"TRUE", true}, {"True", true},
		{"yes", true}, {"YES", true}, {"on", true}, {"On", true},
		{"0", false}, {"false", false}, {"FALSE", false}, {"False", false},
		{"no", false}, {"NO", false}, {"off", false}, {"Off", false},
	}
	for _, c := range cases {
		got, err := ParseBool(c.in)
		assert.NilError(t, err, c.in)
		assert.Equal(t, c.want, got, c.in)
	}
}

func TestParseBool_RejectsInvalid(t *testing.T) {
	for _, in := range []string{"", "2", "truthy", "y", "n", "true ", " false", "*", "tru e"} {
		_, err := ParseBool(in)
		assert.Assert(t, err != nil, "expected error for %q", in)
	}
}

func TestParseInt(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"0", 0}, {"-1", -1}, {"42", 42}, {"007", 7},
		{"9223372036854775807", 9223372036854775807},
	}
	for _, c := range cases {
		got, err := ParseInt(c.in)
		assert.NilError(t, err, c.in)
		assert.Equal(t, c.want, got, c.in)
	}
}

func TestParseInt_RejectsInvalid(t *testing.T) {
	for _, in := range []string{"", "abc", "1.0", "  42", "42 ", "9223372036854775808", "0x10"} {
		_, err := ParseInt(in)
		assert.Assert(t, err != nil, "expected error for %q", in)
	}
}

func TestParseList(t *testing.T) {
	cases := []struct {
		in    string
		delim string
		want  []string
	}{
		{"a,b,c", ",", []string{"a", "b", "c"}},
		{"a b c", " ", []string{"a", "b", "c"}},
		{"deb a; deb b", ";", []string{"deb a", "deb b"}},
		{"a b, c d", ",", []string{"a b", "c d"}},
		{"a,b,", ",", []string{"a", "b"}},
		{"a,,b", ",", []string{"a", "b"}},
		{"  a  ,  b  ", ",", []string{"a", "b"}},
		{"α,β", ",", []string{"α", "β"}},
		{"", ",", nil},
	}
	for _, c := range cases {
		got := ParseList(c.in, c.delim)
		assert.DeepEqual(t, c.want, got)
	}
}

func TestResolveBool_GoesThroughCache(t *testing.T) {
	_installFixture(t, sampleYAML)
	t.Setenv("WS_SERVER_ROOT", "true")
	got, err := ResolveBool("server", "root")
	assert.NilError(t, err)
	assert.Equal(t, true, got)
}

func TestResolveInt_FallsBackToYAMLDefault(t *testing.T) {
	_installFixture(t, sampleYAML)
	got, err := ResolveInt("metrics", "port")
	assert.NilError(t, err)
	assert.Equal(t, int64(9100), got)
}

func TestResolveList_HonorsYAMLDelimiter(t *testing.T) {
	_installFixture(t, sampleYAML)
	t.Setenv("WS_APT_ADDITIONAL_REPOS", "deb a; deb b")
	got, err := ResolveList("apt", "additional_repos", "")
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"deb a", "deb b"}, got)
}

func TestResolveList_OverrideWinsOverYAMLDelimiter(t *testing.T) {
	_installFixture(t, sampleYAML)
	t.Setenv("WS_APT_ADDITIONAL_REPOS", "deb a, deb b")
	got, err := ResolveList("apt", "additional_repos", ",")
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"deb a", "deb b"}, got)
}

func TestCheck(t *testing.T) {
	cases := []struct {
		name      string
		preferred string
		alias     string
		setPref   string
		setAlias  string
		want      CheckState
	}{
		{"PreferredOnly", "WS_NEW", "WS_OLD", "v", "", CheckPreferredSet},
		{"DeprecatedOnly", "WS_NEW", "WS_OLD", "", "v", CheckDeprecatedOnly},
		{"BothSet", "WS_NEW", "WS_OLD", "a", "b", CheckBothSet},
		{"Unset", "WS_NEW", "WS_OLD", "", "", CheckUnset},
		{"PreferredOnlyNoDeprecated", "WS_NEW", "", "v", "", CheckPreferredSet},
		{"UnsetNoDeprecated", "WS_NEW", "", "", "", CheckUnset},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			os.Unsetenv("WS_NEW")
			os.Unsetenv("WS_OLD")
			if c.setPref != "" {
				t.Setenv("WS_NEW", c.setPref)
			}
			if c.setAlias != "" && c.alias != "" {
				t.Setenv(c.alias, c.setAlias)
			}
			got := Check(c.preferred, c.alias)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestDeprecationLine(t *testing.T) {
	assert.Equal(t,
		"Deprecated: [WS_OLD] use [WS_NEW] instead",
		DeprecationLine("WS_OLD", "WS_NEW"),
	)
}

func TestBothSetLine(t *testing.T) {
	assert.Equal(t,
		"Both [WS_OLD] (deprecated) and [WS_NEW] are set\n. Aborting",
		BothSetLine("WS_OLD", "WS_NEW"),
	)
}

func TestResolve_TypePath_TildeExpansion(t *testing.T) {
	_installPathFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_SECRETS_VAULT", "")

	value, err := ResolveKey("WS_SECRETS_VAULT")
	assert.NilError(t, err)
	assert.Equal(t, "/home/kloud/.ws/vault/secrets.yaml", value)
}

func TestResolve_TypePath_BareTilde(t *testing.T) {
	_installPathFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_FEATURES_DIR", "")

	value, err := ResolveKey("WS_FEATURES_DIR")
	assert.NilError(t, err)
	assert.Equal(t, "/home/kloud", value)
}

func TestResolve_TypePath_AbsoluteUnchanged(t *testing.T) {
	_installPathFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_SERVER_SSL_CERT", "")

	value, err := ResolveKey("WS_SERVER_SSL_CERT")
	assert.NilError(t, err)
	assert.Equal(t, "/etc/workspace/ssl/cert.pem", value)
}

func TestResolve_TypePath_MidStringNotExpanded(t *testing.T) {
	_installPathFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_PATHS_MIDSTRING", "")

	value, err := ResolveKey("WS_PATHS_MIDSTRING")
	assert.NilError(t, err)
	assert.Equal(t, "/foo/~/bar", value)
}

func TestResolve_TypePath_NoVarInterpolation(t *testing.T) {
	_installPathFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_PATHS_DOLLARHOME", "")

	value, err := ResolveKey("WS_PATHS_DOLLARHOME")
	assert.NilError(t, err)
	assert.Equal(t, "$HOME/.ws", value)
}

func TestResolve_TypePath_HomeFallback(t *testing.T) {
	_installPathFixture(t)
	t.Setenv("HOME", "")
	t.Setenv("WS_SECRETS_VAULT", "")

	value, err := ResolveKey("WS_SECRETS_VAULT")
	assert.NilError(t, err)
	assert.Equal(t, "/home/kloud/.ws/vault/secrets.yaml", value)
}

func TestResolve_TypePath_HomeFallback_SourceStaysDefault(t *testing.T) {
	_installPathFixture(t)
	t.Setenv("HOME", "")
	t.Setenv("WS_SECRETS_VAULT", "")

	value, source, err := ResolveKeyWithSource("WS_SECRETS_VAULT")
	assert.NilError(t, err)
	assert.Equal(t, "/home/kloud/.ws/vault/secrets.yaml", value)
	assert.Equal(t, SourceDefault, source)
}

func TestResolve_TypePath_EnvOverride_TildeStillExpands(t *testing.T) {
	_installPathFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_SECRETS_VAULT", "~/custom/vault.yaml")

	value, source, err := ResolveKeyWithSource("WS_SECRETS_VAULT")
	assert.NilError(t, err)
	assert.Equal(t, "/home/kloud/custom/vault.yaml", value)
	assert.Equal(t, SourceEnv, source)
}

func TestResolve_TypePath_EnvOverride_AbsolutePassesThrough(t *testing.T) {
	_installPathFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_SECRETS_VAULT", "/custom/path/secrets.yaml")

	value, source, err := ResolveKeyWithSource("WS_SECRETS_VAULT")
	assert.NilError(t, err)
	assert.Equal(t, "/custom/path/secrets.yaml", value)
	assert.Equal(t, SourceEnv, source)
}

func TestResolve_TypePath_OtherUserTildeUnchanged(t *testing.T) {
	_installPathFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_PATHS_OTHERUSER", "")

	value, err := ResolveKey("WS_PATHS_OTHERUSER")
	assert.NilError(t, err)
	assert.Equal(t, "~root/x", value)
}

func TestResolve_TypePath_EmptyEnvExpandsDefault(t *testing.T) {
	_installPathFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_SECRETS_VAULT", "")

	value, source, err := ResolveKeyWithSource("WS_SECRETS_VAULT")
	assert.NilError(t, err)
	assert.Equal(t, "/home/kloud/.ws/vault/secrets.yaml", value)
	assert.Equal(t, SourceDefault, source)
}

func TestResolve_TypePath_EmptyDefaultStaysEmpty(t *testing.T) {
	_installPathFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_SECRETS_MASTER_KEY_FILE", "")

	value, source, err := ResolveKeyWithSource("WS_SECRETS_MASTER_KEY_FILE")
	assert.NilError(t, err)
	assert.Equal(t, "", value)
	assert.Equal(t, SourceDefault, source)
}

func TestResolve_TypePath_ExpandsAfterDeprecatedAlias(t *testing.T) {
	_installPathFixture(t)
	_captureWarnings(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_SECRETS_VAULT", "")
	t.Setenv("WS_VAULT", "~/legacy/vault.yaml")

	value, source, err := ResolveKeyWithSource("WS_SECRETS_VAULT")
	assert.NilError(t, err)
	assert.Equal(t, "/home/kloud/legacy/vault.yaml", value)
	assert.Equal(t, SourceDeprecatedAlias, source)
}

func TestResolveListKey_TypePath_PerElementExpansion(t *testing.T) {
	_installPathFixture(t)
	t.Setenv("HOME", "/home/kloud")
	t.Setenv("WS_FEATURES_EXTRA_PATHS", "")

	items, err := ResolveListKey("WS_FEATURES_EXTRA_PATHS", "")
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"/home/kloud/local/bin", "/opt/bin"}, items)
}

const secretYAML = `
envs:
  auth:
    properties:
      password:
        type: string
        default: null
        secret: true
      password_hashed:
        type: string
        default: null
        secret: true
      github_token:
        type: string
        default: null
        secret: true
  secrets:
    properties:
      master_key:
        type: string
        default: null
        secret: true
  server:
    properties:
      root:
        type: string
        default: /workspace
      ssl_cert:
        type: path
        default: null
        secret: true
      ssl_key:
        type: path
        default: null
        secret: true
  features:
    properties:
      list:
        type: path
        delimiter: ":"
        default: null
        secret: true
deprecated:
  WS_AUTH_PASSWORD_FILE:
    use: WS_AUTH_PASSWORD
    since: 0.3.0
    removed: 0.3.0
    message: tombstone
`

func _installSecretFixture(t *testing.T) {
	t.Helper()
	_installFixture(t, secretYAML)
}

func _newSecretRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("WS__INTERNAL_SECRETS_ROOT", root)
	return root
}

func _writeAt(t *testing.T, path, contents string) {
	t.Helper()
	assert.NilError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NilError(t, os.WriteFile(path, []byte(contents), 0o600))
}

func TestResolve_FilePrefix_ReadsFileContents(t *testing.T) {
	_installSecretFixture(t)
	_newSecretRoot(t)
	pwFile := filepath.Join(t.TempDir(), "pw")
	_writeAt(t, pwFile, "payload\n")
	t.Setenv("WS_AUTH_PASSWORD", "file:"+pwFile)

	value, source, err := ResolveKeyWithSource("WS_AUTH_PASSWORD")
	assert.NilError(t, err)
	assert.Equal(t, "payload", value)
	assert.Equal(t, SourceEnvFile, source)
}

func TestResolve_FilePrefix_TrimsTrailingNewline(t *testing.T) {
	_installSecretFixture(t)
	_newSecretRoot(t)
	pwFile := filepath.Join(t.TempDir(), "pw")
	_writeAt(t, pwFile, "secret\n\n")
	t.Setenv("WS_AUTH_PASSWORD", "file:"+pwFile)

	value, _, err := ResolveKeyWithSource("WS_AUTH_PASSWORD")
	assert.NilError(t, err)
	assert.Equal(t, "secret\n", value)
}

func TestResolve_FilePrefix_PreservesInternalWhitespace(t *testing.T) {
	_installSecretFixture(t)
	_newSecretRoot(t)
	pwFile := filepath.Join(t.TempDir(), "pw")
	_writeAt(t, pwFile, "multi\nline\nvalue\n")
	t.Setenv("WS_AUTH_PASSWORD", "file:"+pwFile)

	value, _, err := ResolveKeyWithSource("WS_AUTH_PASSWORD")
	assert.NilError(t, err)
	assert.Equal(t, "multi\nline\nvalue", value)
}

func TestResolve_FilePrefix_MissingFileErrors(t *testing.T) {
	_installSecretFixture(t)
	_newSecretRoot(t)
	t.Setenv("WS_AUTH_PASSWORD", "file:/no/such/path")

	_, _, err := ResolveKeyWithSource("WS_AUTH_PASSWORD")
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "/no/such/path")
}

func TestResolve_FilePrefix_OnNonSecretPropertyErrors(t *testing.T) {
	_installSecretFixture(t)
	_newSecretRoot(t)
	t.Setenv("WS_SERVER_ROOT", "file:/tmp/x")

	_, _, err := ResolveKeyWithSource("WS_SERVER_ROOT")
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "file: prefix is only valid on secret properties")
	assert.ErrorContains(t, err, "WS_SERVER_ROOT")
}

func TestResolve_FilePrefix_EmptyPathErrors(t *testing.T) {
	_installSecretFixture(t)
	_newSecretRoot(t)
	t.Setenv("WS_AUTH_PASSWORD", "file:")

	_, _, err := ResolveKeyWithSource("WS_AUTH_PASSWORD")
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "file: prefix requires a path")
}

func TestResolve_SecretConventionDefault_FileExists(t *testing.T) {
	_installSecretFixture(t)
	root := _newSecretRoot(t)
	_writeAt(t, filepath.Join(root, "auth/password"), "fromconv\n")
	t.Setenv("WS_AUTH_PASSWORD", "")

	value, source, err := ResolveKeyWithSource("WS_AUTH_PASSWORD")
	assert.NilError(t, err)
	assert.Equal(t, "fromconv", value)
	assert.Equal(t, SourceSecretFileDefault, source)
}

func TestResolve_SecretConventionDefault_FileMissingFallsThroughToYAMLDefault(t *testing.T) {
	_installSecretFixture(t)
	_newSecretRoot(t)
	t.Setenv("WS_AUTH_PASSWORD", "")

	value, source, err := ResolveKeyWithSource("WS_AUTH_PASSWORD")
	assert.NilError(t, err)
	assert.Equal(t, "", value)
	assert.Equal(t, SourceDefault, source)
}

func TestResolve_SecretConventionDefault_NotConsultedForNonSecretProperty(t *testing.T) {
	_installSecretFixture(t)
	root := _newSecretRoot(t)
	_writeAt(t, filepath.Join(root, "server/root"), "/from/convention/path\n")
	t.Setenv("WS_SERVER_ROOT", "")

	value, source, err := ResolveKeyWithSource("WS_SERVER_ROOT")
	assert.NilError(t, err)
	assert.Equal(t, "/workspace", value)
	assert.Equal(t, SourceDefault, source)
}

func TestResolve_DeprecatedTombstone_NoRuntimeEffect(t *testing.T) {
	_installSecretFixture(t)
	_newSecretRoot(t)
	t.Setenv("WS_AUTH_PASSWORD", "")
	t.Setenv("WS_AUTH_PASSWORD_FILE", "/tmp/pw")

	value, source, err := ResolveKeyWithSource("WS_AUTH_PASSWORD")
	assert.NilError(t, err)
	assert.Equal(t, "", value)
	assert.Equal(t, SourceDefault, source)
}

func TestResolve_FilePrefix_AppliesAfterTildeExpansionForTypePath(t *testing.T) {
	_installSecretFixture(t)
	_newSecretRoot(t)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	_writeAt(t, filepath.Join(homeDir, "secrets/server.crt"), "PEM-BODY\n")
	t.Setenv("WS_SERVER_SSL_CERT", "file:~/secrets/server.crt")

	value, source, err := ResolveKeyWithSource("WS_SERVER_SSL_CERT")
	assert.NilError(t, err)
	assert.Equal(t, "PEM-BODY", value)
	assert.Equal(t, SourceEnvFile, source)
}

func TestResolve_SecretConventionDefault_NoLegacyMasterKeyFallback(t *testing.T) {
	_installSecretFixture(t)
	_newSecretRoot(t)
	t.Setenv("WS_SECRETS_MASTER_KEY", "")

	value, source, err := ResolveKeyWithSource("WS_SECRETS_MASTER_KEY")
	assert.NilError(t, err)
	assert.Equal(t, "", value)
	assert.Equal(t, SourceDefault, source)
}

func TestResolve_NilRef_SecretFastPathIsNoOp(t *testing.T) {
	t.Setenv("WS__INTERNAL_ENV_REFERENCE", filepath.Join(t.TempDir(), "missing.yaml"))
	t.Setenv("WS_AUTH_PASSWORD", "file:/tmp/x")

	value, source, err := ResolveKeyWithSource("WS_AUTH_PASSWORD")
	assert.NilError(t, err)
	assert.Equal(t, "file:/tmp/x", value)
	assert.Equal(t, SourceEnv, source)
}

func TestConventionPath_FromRuntimeKey(t *testing.T) {
	cases := []struct {
		runtimeKey string
		want       string
	}{
		{"WS_AUTH_PASSWORD", "/run/secrets/workspace/auth/password"},
		{"WS_AUTH_PASSWORD_HASHED", "/run/secrets/workspace/auth/password_hashed"},
		{"WS_AUTH_GITHUB_TOKEN", "/run/secrets/workspace/auth/github_token"},
		{"WS_SECRETS_MASTER_KEY", "/run/secrets/workspace/secrets/master_key"},
		{"WS_SERVER_SSL_CERT", "/run/secrets/workspace/server/ssl_cert"},
		{"WS_SERVER_SSL_KEY", "/run/secrets/workspace/server/ssl_key"},
	}
	_installSecretFixture(t)
	t.Setenv("WS__INTERNAL_SECRETS_ROOT", defaultSecretConventionRoot)
	ref, err := LoadEnvReference()
	assert.NilError(t, err)
	for _, c := range cases {
		prop := ref.Properties[c.runtimeKey]
		assert.Equal(t, c.want, conventionSecretPath(prop), "key %s", c.runtimeKey)
	}
}

func TestResolveListKey_Secret_HandlesFilePrefix(t *testing.T) {
	_installSecretFixture(t)
	_newSecretRoot(t)
	listFile := filepath.Join(t.TempDir(), "list")
	_writeAt(t, listFile, "a:b:c\n")
	t.Setenv("WS_FEATURES_LIST", "file:"+listFile)

	items, err := ResolveListKey("WS_FEATURES_LIST", "")
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"a", "b", "c"}, items)
}
