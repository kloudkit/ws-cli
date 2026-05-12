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
