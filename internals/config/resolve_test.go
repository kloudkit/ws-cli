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
