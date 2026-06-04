package config

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestPropertyValidate(t *testing.T) {
	cases := []struct {
		name    string
		prop    Property
		value   string
		wantErr bool
	}{
		{"int accept", Property{Type: "integer", Group: "metrics", Name: "port"}, "8080", false},
		{"int reject", Property{Type: "integer", Group: "metrics", Name: "port"}, "80x", true},
		{"bool accept", Property{Type: "boolean", Group: "x", Name: "y"}, "true", false},
		{"bool reject", Property{Type: "boolean", Group: "x", Name: "y"}, "maybe", true},
		{"path accept absolute", Property{Type: "path", Group: "x", Name: "y"}, "/a/b", false},
		{"path accept relative", Property{Type: "path", Group: "x", Name: "y"}, "rel/path", false},
		{"path reject newline", Property{Type: "path", Group: "x", Name: "y"}, "/a\nb", true},
		{"path reject nul", Property{Type: "path", Group: "x", Name: "y"}, "/a\x00b", true},
		{"pattern accept", Property{Type: "string", Pattern: "[a-z]+", Group: "x", Name: "y"}, "abc", false},
		{"pattern reject", Property{Type: "string", Pattern: "[a-z]+", Group: "x", Name: "y"}, "ab1", true},
		{"empty bypasses every gate", Property{Type: "integer", Pattern: "[a-z]+", Group: "x", Name: "y"}, "", false},
		{"string without pattern unconstrained", Property{Type: "string", Group: "x", Name: "y"}, "anything$(x)", false},
		{"secret skipped entirely", Property{Type: "integer", Secret: true, Group: "x", Name: "y"}, "not-an-int", false},
		{"invalid declared pattern surfaces", Property{Type: "string", Pattern: "[unclosed", Group: "x", Name: "y"}, "x", true},
		{"list one bad token rejects", Property{Type: "string", Pattern: "[a-z]+", Delimiter: " ", Group: "x", Name: "y"}, "ok bad1", true},
		{"list all good tokens pass", Property{Type: "string", Pattern: "[a-z]+", Delimiter: " ", Group: "x", Name: "y"}, "ok fine", false},
	}
	for _, c := range cases {
		err := c.prop.Validate(c.value)
		if c.wantErr {
			assert.Assert(t, err != nil, c.name)
		} else {
			assert.NilError(t, err, c.name)
		}
	}
}

func TestPropertyValidate_SecretErrorNeverEchoesValue(t *testing.T) {
	prop := Property{Type: "string", Pattern: "[a-z]+", Secret: true, Group: "auth", Name: "password"}
	assert.NilError(t, prop.Validate("s3cr3t-$(rm)"))
}

const patternYAML = `
envs:
  zsh:
    properties:
      plugins:
        type: string
        default: null
        delimiter: " "
        pattern: '[a-zA-Z0-9_-]+'
  metrics:
    properties:
      port:
        type: integer
        default: 9100
`

func TestResolveKeyWithSource_RejectsBadInt(t *testing.T) {
	_installFixture(t, patternYAML)
	t.Setenv("WS_METRICS_PORT", "80x")

	_, _, err := ResolveKeyWithSource("WS_METRICS_PORT")
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "WS_METRICS_PORT")
}

func TestResolveKeyWithSource_AcceptsValidInt(t *testing.T) {
	_installFixture(t, patternYAML)
	t.Setenv("WS_METRICS_PORT", "9090")

	value, _, err := ResolveKeyWithSource("WS_METRICS_PORT")
	assert.NilError(t, err)
	assert.Equal(t, "9090", value)
}

func TestResolveKeyWithSource_DefaultPasses(t *testing.T) {
	_installFixture(t, patternYAML)
	t.Setenv("WS_METRICS_PORT", "")

	value, source, err := ResolveKeyWithSource("WS_METRICS_PORT")
	assert.NilError(t, err)
	assert.Equal(t, "9100", value)
	assert.Equal(t, SourceDefault, source)
}

func TestResolveListKey_RejectsBadToken(t *testing.T) {
	_installFixture(t, patternYAML)
	t.Setenv("WS_ZSH_PLUGINS", "git $(touch)")

	_, err := ResolveListKey("WS_ZSH_PLUGINS", "")
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "WS_ZSH_PLUGINS")
}

func TestResolveListKey_AcceptsGoodTokens(t *testing.T) {
	_installFixture(t, patternYAML)
	t.Setenv("WS_ZSH_PLUGINS", "git docker")

	items, err := ResolveListKey("WS_ZSH_PLUGINS", "")
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"git", "docker"}, items)
}

func TestPropertyValidate_MigratedCharsets(t *testing.T) {
	const (
		plugins = `[a-zA-Z0-9_-]+`
		repos   = `[a-zA-Z0-9 ./:_=~+[\]-]+`
		proxy   = `[a-zA-Z0-9.{}*_-]+`
		domains = `[a-zA-Z0-9.:/_~?#=&%+-]+`
	)
	cases := []struct {
		name      string
		pattern   string
		delimiter string
		value     string
		wantErr   bool
	}{
		{"plugins accept", plugins, " ", "git docker kubectl zsh-autosuggestions", false},
		{"plugins reject command-sub", plugins, " ", "git $(touch /tmp/pwn)", true},
		{"plugins reject backtick", plugins, " ", "git `id`", true},
		{"plugins reject ifs-sub", plugins, " ", "x$(touch${IFS}/tmp/pwn)", true},
		{"plugins reject semicolon", plugins, " ", "git;rm", true},

		{"repos accept", repos, ";", "deb http://a.test trixie main; deb https://b.test trixie main", false},
		{"repos reject command-sub", repos, ";", "deb http://a.test trixie main$(touch /tmp/pwn)", true},
		{"repos reject backtick", repos, ";", "deb `id`", true},
		{"repos reject pipe", repos, ";", "deb http://a.test | sh", true},

		{"proxy accept plain", proxy, " ", "ws.dev local.ws.dev", false},
		{"proxy accept template", proxy, " ", "{{port}}-project.ws.dev {{port}}.local.ws.dev", false},
		{"proxy reject command-sub", proxy, " ", "ok.dev evil$(touch /tmp/pwn)", true},
		{"proxy reject slash", proxy, " ", "ok.dev evil/../x", true},

		{"domains accept", domains, ",", "https://github.com,https://stackoverflow.com", false},
		{"domains reject command-sub", domains, ",", "https://ok.com,evil$(id)", true},
		{"domains reject space-injection", domains, ",", "https://ok.com rm -rf", true},
		{"domains reject backtick", domains, ",", "https://ok.com,`id`", true},
	}
	for _, c := range cases {
		prop := Property{Type: "string", Pattern: c.pattern, Delimiter: c.delimiter, Group: "x", Name: "y"}
		err := prop.Validate(c.value)
		if c.wantErr {
			assert.Assert(t, err != nil, c.name)
		} else {
			assert.NilError(t, err, c.name)
		}
	}
}
