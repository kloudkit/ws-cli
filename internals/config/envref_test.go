package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"gotest.tools/v3/assert"
)

func _prop(name, body string) string {
	return "envs:\n  g:\n    properties:\n      " + name + ":\n" + body
}

func TestParseEnvReference(t *testing.T) {
	cases := []struct {
		name, yaml, key string
		check           func(t *testing.T, p Property)
		err             []string
	}{
		{name: "PathNullDefault", yaml: _prop("p", "        type: path\n        default: null\n"), key: "WS_G_P", check: func(t *testing.T, p Property) {
			assert.Equal(t, "path", p.Type)
			assert.Assert(t, p.Default == nil)
		}},
		{name: "PathAbsoluteDefault", yaml: _prop("p", "        type: path\n        default: /var/lib/workspace\n"), key: "WS_G_P", check: func(t *testing.T, p Property) {
			assert.Equal(t, "/var/lib/workspace", *p.Default)
		}},
		{name: "PathTildeDefault", yaml: _prop("p", "        type: path\n        default: \"~/.ws/token\"\n"), key: "WS_G_P", check: func(t *testing.T, p Property) {
			assert.Equal(t, "~/.ws/token", *p.Default)
		}},
		{name: "IntegerDefaultStringified", yaml: _prop("p", "        type: integer\n        default: 9100\n"), key: "WS_G_P", check: func(t *testing.T, p Property) {
			assert.Equal(t, "9100", *p.Default)
		}},
		{name: "SecretTrue", yaml: _prop("p", "        type: string\n        secret: true\n"), key: "WS_G_P", check: func(t *testing.T, p Property) {
			assert.Equal(t, true, p.Secret)
		}},
		{name: "SecretDefaultsFalse", yaml: _prop("p", "        type: string\n        secret: false\n"), key: "WS_G_P", check: func(t *testing.T, p Property) {
			assert.Equal(t, false, p.Secret)
		}},
		{name: "PathBoolDefaultRejected", yaml: _prop("p", "        type: path\n        default: false\n"), err: []string{"type [path]", "bool"}},
		{name: "PathIntDefaultRejected", yaml: _prop("p", "        type: path\n        default: 42\n"), err: []string{"type [path]", "int"}},
		{name: "UnknownTypeRejected", yaml: _prop("p", "        type: bogus\n"), err: []string{"[g.p]: unknown type [bogus]"}},
		{name: "SecretFileDefaultRejected", yaml: _prop("p", "        type: string\n        default: \"file:/etc/foo\"\n        secret: true\n"), err: []string{"secret", "[file:]"}},
		{name: "SecretBoolDefaultRejected", yaml: _prop("p", "        type: string\n        default: false\n        secret: true\n"), err: []string{"secret requires default"}},
		{name: "DeprecationCycleRejected", yaml: "deprecated:\n  WS_A: {use: WS_B}\n  WS_B: {use: WS_A}\n", err: []string{"deprecation cycle"}},
		{name: "EmptyEnvs", yaml: "envs: {}\ndeprecated: {}\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ref, err := parseEnvReference([]byte(c.yaml))
			for _, want := range c.err {
				assert.ErrorContains(t, err, want)
			}
			if c.err != nil {
				return
			}
			assert.NilError(t, err)
			if c.check != nil {
				c.check(t, ref.Properties[c.key])
			}
		})
	}
}

func TestParseEnvReference_Descriptions(t *testing.T) {
	ref, err := parseEnvReference([]byte(sampleYAML))
	assert.NilError(t, err)
	for key, want := range map[string][2]string{
		"WS_APT_ADDITIONAL_PACKAGES":      {"Additional APT packages installed during startup.", "Accepts a **space-delimited** package list.\n"},
		"WS_METRICS_PORT":                 {"Port on which the metrics endpoint listens.", "The metrics server exposes a `/` endpoint on this port.\n"},
		"WS_SERVER_ROOT":                  {"Root directory for the workspace.", ""},
		"WS_FEATURES_ADDITIONAL_FEATURES": {"", ""},
	} {
		prop := ref.Properties[key]
		assert.DeepEqual(t, want, [2]string{prop.Description, prop.LongDescription})
	}
}

func TestParseEnvReference_Aliases(t *testing.T) {
	ref, err := parseEnvReference([]byte(`
deprecated:
  WS_A: {use: WS_B}
  WS_B: {use: WS_C}
  WS_GONE: {use: WS_C, removed: 0.3.0}
`))
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"WS_A", "WS_B"}, slices.Sorted(slices.Values(ref.AliasesByPreferred["WS_C"])))
}

func TestLoadEnvReference(t *testing.T) {
	t.Setenv("WS__INTERNAL_ENV_REFERENCE", filepath.Join(t.TempDir(), "missing.yaml"))
	_, err := LoadEnvReference()
	assert.ErrorContains(t, err, "cannot read")

	dir := t.TempDir()
	for name, content := range map[string]string{"a": _prop("root", "        type: string\n"), "b": _prop("port", "        type: integer\n")} {
		assert.NilError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
	for name, key := range map[string]string{"a": "WS_G_ROOT", "b": "WS_G_PORT"} {
		t.Setenv("WS__INTERNAL_ENV_REFERENCE", filepath.Join(dir, name))
		ref, err := LoadEnvReference()
		assert.NilError(t, err)
		assert.Equal(t, 1, len(ref.Properties))
		_, ok := ref.Properties[key]
		assert.Assert(t, ok, "load must re-read the re-pointed reference")
	}
}

func TestPropertyValidate(t *testing.T) {
	const (
		plugins = `[a-zA-Z0-9_-]+`
		repos   = `[a-zA-Z0-9 ./:_=~+[\]-]+`
		proxy   = `[a-zA-Z0-9.{}*_-]+`
		domains = `[a-zA-Z0-9.:/_~?#=&%+-]+`
	)
	str := func(pattern, delim string) Property {
		return Property{Type: "string", Pattern: pattern, Delimiter: delim, Group: "x", Name: "y"}
	}
	cases := []struct {
		name  string
		prop  Property
		value string
		ok    bool
	}{
		{"int accept", Property{Type: "integer"}, "8080", true},
		{"int reject", Property{Type: "integer"}, "80x", false},
		{"bool accept", Property{Type: "boolean"}, "true", true},
		{"bool reject", Property{Type: "boolean"}, "maybe", false},
		{"path absolute", Property{Type: "path"}, "/a/b", true},
		{"path relative", Property{Type: "path"}, "rel/path", true},
		{"path newline", Property{Type: "path"}, "/a\nb", false},
		{"path nul", Property{Type: "path"}, "/a\x00b", false},
		{"pattern accept", str("[a-z]+", ""), "abc", true},
		{"pattern reject", str("[a-z]+", ""), "ab1", false},
		{"empty bypasses gates", Property{Type: "integer", Pattern: "[a-z]+"}, "", true},
		{"string unconstrained", Property{Type: "string"}, "anything$(x)", true},
		{"secret skipped", Property{Type: "integer", Pattern: "[a-z]+", Secret: true}, "s3cr3t-$(rm)", true},
		{"invalid declared pattern", str("[unclosed", ""), "x", false},
		{"list one bad token", str("[a-z]+", " "), "ok bad1", false},
		{"list all good", str("[a-z]+", " "), "ok fine", true},

		{"plugins accept", str(plugins, " "), "git docker kubectl zsh-autosuggestions", true},
		{"plugins command-sub", str(plugins, " "), "git $(touch /tmp/pwn)", false},
		{"plugins backtick", str(plugins, " "), "git `id`", false},
		{"plugins ifs-sub", str(plugins, " "), "x$(touch${IFS}/tmp/pwn)", false},
		{"plugins semicolon", str(plugins, " "), "git;rm", false},
		{"repos accept", str(repos, ";"), "deb http://a.test trixie main; deb https://b.test trixie main", true},
		{"repos command-sub", str(repos, ";"), "deb http://a.test trixie main$(touch /tmp/pwn)", false},
		{"repos backtick", str(repos, ";"), "deb `id`", false},
		{"repos pipe", str(repos, ";"), "deb http://a.test | sh", false},
		{"proxy plain", str(proxy, " "), "ws.dev local.ws.dev", true},
		{"proxy template", str(proxy, " "), "{{port}}-project.ws.dev {{port}}.local.ws.dev", true},
		{"proxy command-sub", str(proxy, " "), "ok.dev evil$(touch /tmp/pwn)", false},
		{"proxy slash", str(proxy, " "), "ok.dev evil/../x", false},
		{"domains accept", str(domains, ","), "https://github.com,https://stackoverflow.com", true},
		{"domains command-sub", str(domains, ","), "https://ok.com,evil$(id)", false},
		{"domains space-injection", str(domains, ","), "https://ok.com rm -rf", false},
		{"domains backtick", str(domains, ","), "https://ok.com,`id`", false},
	}
	for _, c := range cases {
		err := c.prop.Validate(c.value)
		if c.ok {
			assert.NilError(t, err, c.name)
		} else {
			assert.Assert(t, err != nil, c.name)
		}
	}
}

func TestParseBool(t *testing.T) {
	for _, in := range []string{"1", "true", "TRUE", "True", "yes", "YES", "on", "On"} {
		got, err := ParseBool(in)
		assert.NilError(t, err, in)
		assert.Assert(t, got, in)
	}
	for _, in := range []string{"0", "false", "FALSE", "False", "no", "NO", "off", "Off"} {
		got, err := ParseBool(in)
		assert.NilError(t, err, in)
		assert.Assert(t, !got, in)
	}
	for _, in := range []string{"", "2", "truthy", "y", "n", "true ", " false", "*", "tru e"} {
		_, err := ParseBool(in)
		assert.ErrorContains(t, err, "not a boolean", in)
	}
}

func TestParseInt(t *testing.T) {
	for in, want := range map[string]int64{"0": 0, "-1": -1, "42": 42, "007": 7, "9223372036854775807": 9223372036854775807} {
		got, err := ParseInt(in)
		assert.NilError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	for _, in := range []string{"", "abc", "1.0", "  42", "42 ", "9223372036854775808", "0x10"} {
		_, err := ParseInt(in)
		assert.ErrorContains(t, err, "not an integer", in)
	}
}

func TestParseList(t *testing.T) {
	cases := []struct {
		in, delim string
		want      []string
	}{
		{"a,b,c", ",", []string{"a", "b", "c"}},
		{"a b c", "", []string{"a", "b", "c"}},
		{"deb a; deb b", ";", []string{"deb a", "deb b"}},
		{"a b, c d", ",", []string{"a b", "c d"}},
		{"a,b,", ",", []string{"a", "b"}},
		{"a,,b", ",", []string{"a", "b"}},
		{"  a  ,  b  ", ",", []string{"a", "b"}},
		{"α,β", ",", []string{"α", "β"}},
		{"", ",", nil},
	}
	for _, c := range cases {
		assert.DeepEqual(t, c.want, ParseList(c.in, c.delim))
	}
}
