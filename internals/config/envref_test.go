package config

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParse_TypePath_NullDefault(t *testing.T) {
	yamlData := `
envs:
  auth:
    properties:
      github_token_file:
        type: path
        default: null
`
	r, err := parseEnvReference([]byte(yamlData))
	assert.NilError(t, err)
	prop := r.Properties["WS_AUTH_GITHUB_TOKEN_FILE"]
	assert.Equal(t, "path", prop.Type)
	assert.Assert(t, prop.Default == nil)
}

func TestParse_TypePath_AbsoluteStringDefault(t *testing.T) {
	yamlData := `
envs:
  server:
    properties:
      root_dir:
        type: path
        default: "/var/lib/workspace"
`
	r, err := parseEnvReference([]byte(yamlData))
	assert.NilError(t, err)
	prop := r.Properties["WS_SERVER_ROOT_DIR"]
	assert.Equal(t, "path", prop.Type)
	assert.Equal(t, "/var/lib/workspace", *prop.Default)
}

func TestParse_TypePath_TildeStringDefault(t *testing.T) {
	yamlData := `
envs:
  secrets:
    properties:
      vault:
        type: path
        default: "~/.ws/vault/secrets.yaml"
`
	r, err := parseEnvReference([]byte(yamlData))
	assert.NilError(t, err)
	prop := r.Properties["WS_SECRETS_VAULT"]
	assert.Equal(t, "path", prop.Type)
	assert.Equal(t, "~/.ws/vault/secrets.yaml", *prop.Default)
}

func TestParse_TypePath_BoolDefaultRejected(t *testing.T) {
	yamlData := `
envs:
  features:
    properties:
      dir:
        type: path
        default: false
`
	_, err := parseEnvReference([]byte(yamlData))
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "path")
}

func TestParse_TypePath_IntDefaultRejected(t *testing.T) {
	yamlData := `
envs:
  logging:
    properties:
      dir:
        type: path
        default: 42
`
	_, err := parseEnvReference([]byte(yamlData))
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "path")
}

func TestParse_SecretTrue_AcceptsBoolean(t *testing.T) {
	yamlData := `
envs:
  auth:
    properties:
      password:
        type: string
        default: null
        secret: true
`
	r, err := parseEnvReference([]byte(yamlData))
	assert.NilError(t, err)
	prop := r.Properties["WS_AUTH_PASSWORD"]
	assert.Equal(t, true, prop.Secret)
}

func TestParse_SecretFalse_DefaultsToFalse(t *testing.T) {
	yamlData := `
envs:
  server:
    properties:
      root:
        type: string
        default: /workspace
      explicit:
        type: string
        default: null
        secret: false
`
	r, err := parseEnvReference([]byte(yamlData))
	assert.NilError(t, err)
	assert.Equal(t, false, r.Properties["WS_SERVER_ROOT"].Secret)
	assert.Equal(t, false, r.Properties["WS_SERVER_EXPLICIT"].Secret)
}

func TestParse_SecretTrue_RejectsLiteralFilePrefixDefault(t *testing.T) {
	yamlData := `
envs:
  auth:
    properties:
      password:
        type: string
        default: "file:/etc/foo"
        secret: true
`
	_, err := parseEnvReference([]byte(yamlData))
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "secret")
	assert.ErrorContains(t, err, "file:")
}

func TestParse_SecretTrue_RejectsBoolDefault(t *testing.T) {
	yamlData := `
envs:
  auth:
    properties:
      password:
        type: string
        default: false
        secret: true
`
	_, err := parseEnvReference([]byte(yamlData))
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "secret")
}

func TestParse_DeprecatedTombstone_NotRegisteredAsAlias(t *testing.T) {
	yamlData := `
envs:
  auth:
    properties:
      password:
        type: string
        default: null
        secret: true
deprecated:
  WS_AUTH_PASSWORD_FILE:
    use: WS_AUTH_PASSWORD
    since: 0.3.0
    removed: 0.3.0
    message: tombstone
  WS_OLD_ACTIVE:
    use: WS_AUTH_PASSWORD
`
	r, err := parseEnvReference([]byte(yamlData))
	assert.NilError(t, err)
	aliases := r.AliasesByPreferred["WS_AUTH_PASSWORD"]
	for _, a := range aliases {
		assert.Assert(t, a != "WS_AUTH_PASSWORD_FILE", "tombstone unexpectedly registered: %v", aliases)
	}
	found := false
	for _, a := range aliases {
		if a == "WS_OLD_ACTIVE" {
			found = true
		}
	}
	assert.Assert(t, found, "non-tombstone alias missing from AliasesByPreferred: %v", aliases)
}
