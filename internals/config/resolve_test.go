package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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
  zsh:
    properties:
      plugins:
        type: string
        default: null
        delimiter: " "
        pattern: '[a-zA-Z0-9_-]+'
deprecated:
  WS_PORT:
    use: WS_SERVER_PORT
  WS_OLD_NOREPLACE:
    message: removed without replacement
`

const pathYAML = `
envs:
  secrets:
    properties:
      token_file:
        type: path
        default: "~/.ws/secrets/token"
      master_key_file:
        type: path
        default: null
  server:
    properties:
      ssl_cert:
        type: path
        default: "/etc/workspace/ssl/cert.pem"
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
  WS_TOKEN_FILE:
    use: WS_SECRETS_TOKEN_FILE
`

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

func _installFixture(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "env.reference.yaml")
	if content != "" {
		assert.NilError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	t.Setenv("WS__INTERNAL_ENV_REFERENCE", path)
}

func _captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	original := SetDeprecationWriter(buf)
	t.Cleanup(func() { SetDeprecationWriter(original) })
	ResetWarnedAliases()
	return buf
}

func _writeAt(t *testing.T, path, contents string) {
	t.Helper()
	assert.NilError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NilError(t, os.WriteFile(path, []byte(contents), 0o600))
}

type _env map[string]string

type resolveCase struct {
	name   string
	yaml   string
	env    _env
	files  _env
	key    string
	want   string
	source ResolveSource
	err    string
	warn   string
}

func _runResolve(t *testing.T, c resolveCase, fn func(key string) (string, ResolveSource, error)) {
	t.Helper()
	_installFixture(t, c.yaml)
	tmp, root := t.TempDir(), t.TempDir()
	t.Setenv("WS__INTERNAL_SECRETS_ROOT", root)
	t.Setenv("HOME", "/home/kloud")
	warnings := _captureWarnings(t)
	expand := strings.NewReplacer("{tmp}", tmp, "{root}", root).Replace
	for path, contents := range c.files {
		_writeAt(t, expand(path), contents)
	}
	for key, value := range c.env {
		t.Setenv(key, expand(value))
	}

	value, source, err := fn(c.key)
	if c.err != "" {
		assert.ErrorContains(t, err, c.err)
		return
	}
	assert.NilError(t, err)
	assert.Equal(t, c.want, value)
	assert.Equal(t, c.source, source)
	assert.Equal(t, c.warn, warnings.String())
}

func TestResolveKeyWithSource(t *testing.T) {
	cases := []resolveCase{
		{name: "EnvWinsOverDefault", yaml: sampleYAML, env: _env{"WS_SERVER_ROOT": "/custom"}, key: "WS_SERVER_ROOT", want: "/custom", source: SourceEnv},
		{name: "UnsetReturnsDefault", yaml: sampleYAML, key: "WS_SERVER_ROOT", want: "/workspace", source: SourceDefault},
		{name: "EmptyEnvReturnsDefault", yaml: sampleYAML, env: _env{"WS_SERVER_ROOT": ""}, key: "WS_SERVER_ROOT", want: "/workspace", source: SourceDefault},
		{name: "NullDefaultReturnsEmpty", yaml: sampleYAML, env: _env{"WS_FEATURES_ADDITIONAL_FEATURES": ""}, key: "WS_FEATURES_ADDITIONAL_FEATURES", source: SourceDefault},
		{name: "UnknownKeyReturnsEmpty", yaml: sampleYAML, env: _env{"WS_NOT_DECLARED": ""}, key: "WS_NOT_DECLARED", source: SourceDefault},
		{name: "DeprecatedAliasWarns", yaml: sampleYAML, env: _env{"WS_PORT": "8888"}, key: "WS_SERVER_PORT", want: "8888", source: SourceDeprecatedAlias, warn: "Deprecated: [WS_PORT] use [WS_SERVER_PORT] instead\n"},
		{name: "BothSetPrefersPreferred", yaml: sampleYAML, env: _env{"WS_SERVER_PORT": "9999", "WS_PORT": "8888"}, key: "WS_SERVER_PORT", want: "9999", source: SourceEnv},
		{name: "RejectsBadInt", yaml: sampleYAML, env: _env{"WS_METRICS_PORT": "80x"}, key: "WS_METRICS_PORT", err: "WS_METRICS_PORT"},
		{name: "AcceptsValidInt", yaml: sampleYAML, env: _env{"WS_METRICS_PORT": "9090"}, key: "WS_METRICS_PORT", want: "9090", source: SourceEnv},
		{name: "IntDefaultPasses", yaml: sampleYAML, key: "WS_METRICS_PORT", want: "9100", source: SourceDefault},

		{name: "PathTildeDefault", yaml: pathYAML, key: "WS_SECRETS_TOKEN_FILE", want: "/home/kloud/.ws/secrets/token", source: SourceDefault},
		{name: "PathBareTilde", yaml: pathYAML, key: "WS_FEATURES_DIR", want: "/home/kloud", source: SourceDefault},
		{name: "PathAbsoluteUnchanged", yaml: pathYAML, key: "WS_SERVER_SSL_CERT", want: "/etc/workspace/ssl/cert.pem", source: SourceDefault},
		{name: "PathMidStringNotExpanded", yaml: pathYAML, key: "WS_PATHS_MIDSTRING", want: "/foo/~/bar", source: SourceDefault},
		{name: "PathNoVarInterpolation", yaml: pathYAML, key: "WS_PATHS_DOLLARHOME", want: "$HOME/.ws", source: SourceDefault},
		{name: "PathOtherUserTildeUnchanged", yaml: pathYAML, key: "WS_PATHS_OTHERUSER", want: "~root/x", source: SourceDefault},
		{name: "PathHomeFallback", yaml: pathYAML, env: _env{"HOME": ""}, key: "WS_SECRETS_TOKEN_FILE", want: "/home/kloud/.ws/secrets/token", source: SourceDefault},
		{name: "PathEnvTildeExpands", yaml: pathYAML, env: _env{"WS_SECRETS_TOKEN_FILE": "~/custom/token"}, key: "WS_SECRETS_TOKEN_FILE", want: "/home/kloud/custom/token", source: SourceEnv},
		{name: "PathEnvAbsolutePassesThrough", yaml: pathYAML, env: _env{"WS_SECRETS_TOKEN_FILE": "/custom/path/secrets.yaml"}, key: "WS_SECRETS_TOKEN_FILE", want: "/custom/path/secrets.yaml", source: SourceEnv},
		{name: "PathEmptyDefaultStaysEmpty", yaml: pathYAML, key: "WS_SECRETS_MASTER_KEY_FILE", source: SourceDefault},
		{name: "PathExpandsAfterAlias", yaml: pathYAML, env: _env{"WS_TOKEN_FILE": "~/legacy/token"}, key: "WS_SECRETS_TOKEN_FILE", want: "/home/kloud/legacy/token", source: SourceDeprecatedAlias, warn: "Deprecated: [WS_TOKEN_FILE] use [WS_SECRETS_TOKEN_FILE] instead\n"},

		{name: "FilePrefixTrimsOneNewline", yaml: secretYAML, files: _env{"{tmp}/pw": "payload\n"}, env: _env{"WS_AUTH_PASSWORD": "file:{tmp}/pw"}, key: "WS_AUTH_PASSWORD", want: "payload", source: SourceEnvFile},
		{name: "FilePrefixKeepsInnerNewline", yaml: secretYAML, files: _env{"{tmp}/pw": "secret\n\n"}, env: _env{"WS_AUTH_PASSWORD": "file:{tmp}/pw"}, key: "WS_AUTH_PASSWORD", want: "secret\n", source: SourceEnvFile},
		{name: "FilePrefixMultiline", yaml: secretYAML, files: _env{"{tmp}/pw": "multi\nline\nvalue\n"}, env: _env{"WS_AUTH_PASSWORD": "file:{tmp}/pw"}, key: "WS_AUTH_PASSWORD", want: "multi\nline\nvalue", source: SourceEnvFile},
		{name: "FilePrefixMissingFile", yaml: secretYAML, env: _env{"WS_AUTH_PASSWORD": "file:/no/such/path"}, key: "WS_AUTH_PASSWORD", err: "/no/such/path"},
		{name: "FilePrefixOnNonSecret", yaml: secretYAML, env: _env{"WS_SERVER_ROOT": "file:/tmp/x"}, key: "WS_SERVER_ROOT", err: "file: prefix is only valid on secret properties [WS_SERVER_ROOT]"},
		{name: "FilePrefixEmptyPath", yaml: secretYAML, env: _env{"WS_AUTH_PASSWORD": "file:"}, key: "WS_AUTH_PASSWORD", err: "file: prefix requires a path"},
		{name: "FilePrefixTildeForPath", yaml: secretYAML, files: _env{"{tmp}/secrets/server.crt": "PEM-BODY\n"}, env: _env{"HOME": "{tmp}", "WS_SERVER_SSL_CERT": "file:~/secrets/server.crt"}, key: "WS_SERVER_SSL_CERT", want: "PEM-BODY", source: SourceEnvFile},
		{name: "ConventionMountExists", yaml: secretYAML, files: _env{"{root}/auth/password": "fromconv\n"}, key: "WS_AUTH_PASSWORD", want: "fromconv", source: SourceSecretFileDefault},
		{name: "ConventionMountMissing", yaml: secretYAML, key: "WS_AUTH_PASSWORD", source: SourceDefault},
		{name: "ConventionIgnoredForNonSecret", yaml: secretYAML, files: _env{"{root}/server/root": "/from/convention\n"}, key: "WS_SERVER_ROOT", want: "/workspace", source: SourceDefault},
		{name: "NoLegacyMasterKeyFallback", yaml: secretYAML, key: "WS_SECRETS_MASTER_KEY", source: SourceDefault},
		{name: "TombstoneHasNoEffect", yaml: secretYAML, env: _env{"WS_AUTH_PASSWORD_FILE": "/tmp/pw"}, key: "WS_AUTH_PASSWORD", source: SourceDefault},
		{name: "MissingReferencePassesEnvThrough", env: _env{"WS_AUTH_PASSWORD": "file:/tmp/x"}, key: "WS_AUTH_PASSWORD", want: "file:/tmp/x", source: SourceEnv},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { _runResolve(t, c, ResolveKeyWithSource) })
	}
}

func TestResolveListKey(t *testing.T) {
	cases := []struct {
		resolveCase
		delim string
		items []string
	}{
		{resolveCase{name: "HonorsYAMLDelimiter", yaml: sampleYAML, env: _env{"WS_APT_ADDITIONAL_REPOS": "deb a; deb b"}, key: "WS_APT_ADDITIONAL_REPOS"}, "", []string{"deb a", "deb b"}},
		{resolveCase{name: "OverrideWinsOverYAMLDelimiter", yaml: sampleYAML, env: _env{"WS_APT_ADDITIONAL_REPOS": "deb a, deb b"}, key: "WS_APT_ADDITIONAL_REPOS"}, ",", []string{"deb a", "deb b"}},
		{resolveCase{name: "AcceptsGoodTokens", yaml: sampleYAML, env: _env{"WS_ZSH_PLUGINS": "git docker"}, key: "WS_ZSH_PLUGINS"}, "", []string{"git", "docker"}},
		{resolveCase{name: "RejectsBadToken", yaml: sampleYAML, env: _env{"WS_ZSH_PLUGINS": "git $(touch)"}, key: "WS_ZSH_PLUGINS", err: "WS_ZSH_PLUGINS"}, "", nil},
		{resolveCase{name: "PathPerElementExpansion", yaml: pathYAML, key: "WS_FEATURES_EXTRA_PATHS"}, "", []string{"/home/kloud/local/bin", "/opt/bin"}},
		{resolveCase{name: "SecretFilePrefix", yaml: secretYAML, files: _env{"{tmp}/list": "a:b:c\n"}, env: _env{"WS_FEATURES_LIST": "file:{tmp}/list"}, key: "WS_FEATURES_LIST"}, "", []string{"a", "b", "c"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_runResolve(t, c.resolveCase, func(key string) (string, ResolveSource, error) {
				items, err := ResolveListKey(key, c.delim)
				if err == nil {
					assert.DeepEqual(t, c.items, items)
				}
				return "", 0, err
			})
		})
	}
}

func TestResolve_DeprecatedWarnEmittedOnce(t *testing.T) {
	_installFixture(t, sampleYAML)
	buf := _captureWarnings(t)
	t.Setenv("WS_PORT", "8888")
	for range 3 {
		v, err := Resolve("server", "port")
		assert.NilError(t, err)
		assert.Equal(t, "8888", v)
	}
	assert.Equal(t, 1, bytes.Count(buf.Bytes(), []byte("Deprecated:")))
}

func TestResolveInt_FallsBackToYAMLDefault(t *testing.T) {
	_installFixture(t, sampleYAML)
	got, err := ResolveInt("metrics", "port")
	assert.NilError(t, err)
	assert.Equal(t, int64(9100), got)
}

func TestLookupProperty(t *testing.T) {
	_installFixture(t, sampleYAML)
	for key, want := range map[string]bool{"WS_SERVER_ROOT": true, "WS_NOT_DECLARED": false, "WS__INTERNAL_ENV_REFERENCE": false} {
		prop, ok, err := LookupProperty(key)
		assert.NilError(t, err)
		assert.Equal(t, want, ok, key)
		if want {
			assert.Equal(t, "string", prop.Type)
		}
	}
}

func TestResolveSourceLabel(t *testing.T) {
	assert.DeepEqual(t,
		[]string{"process", "alias", "declared", "file", "mount", ""},
		[]string{SourceEnv.Label(), SourceDeprecatedAlias.Label(), SourceDefault.Label(), SourceEnvFile.Label(), SourceSecretFileDefault.Label(), ResolveSource(99).Label()},
	)
}

func TestSecretConventionPath(t *testing.T) {
	_installFixture(t, secretYAML)
	t.Setenv("WS__INTERNAL_SECRETS_ROOT", defaultSecretConventionRoot)
	ref, err := LoadEnvReference()
	assert.NilError(t, err)
	for _, key := range []string{"WS_AUTH_PASSWORD", "WS_AUTH_PASSWORD_HASHED", "WS_AUTH_GITHUB_TOKEN", "WS_SECRETS_MASTER_KEY", "WS_SERVER_SSL_CERT", "WS_SERVER_SSL_KEY"} {
		prop := ref.Properties[key]
		want := "/run/secrets/workspace/" + strings.ToLower(strings.Replace(strings.TrimPrefix(key, "WS_"), "_", "/", 1))
		assert.Equal(t, want, SecretConventionPath(prop.Group, prop.Name), key)
	}
}

func TestCheck(t *testing.T) {
	cases := []struct {
		name, alias, setPref, setAlias string
		want                           CheckState
	}{
		{"PreferredOnly", "WS_OLD", "v", "", CheckPreferredSet},
		{"DeprecatedOnly", "WS_OLD", "", "v", CheckDeprecatedOnly},
		{"BothSet", "WS_OLD", "a", "b", CheckBothSet},
		{"Unset", "WS_OLD", "", "", CheckUnset},
		{"PreferredOnlyNoDeprecated", "", "v", "", CheckPreferredSet},
		{"UnsetNoDeprecated", "", "", "", CheckUnset},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("WS_NEW", c.setPref)
			t.Setenv("WS_OLD", c.setAlias)
			assert.Equal(t, c.want, Check("WS_NEW", c.alias))
		})
	}
}

func TestWarningLines(t *testing.T) {
	assert.Equal(t, "Deprecated: [WS_OLD] use [WS_NEW] instead", DeprecationLine("WS_OLD", "WS_NEW"))
	assert.Equal(t, "Both [WS_OLD] (deprecated) and [WS_NEW] are set. Aborting", BothSetLine("WS_OLD", "WS_NEW"))
}
