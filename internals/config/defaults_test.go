package config

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestConstants(t *testing.T) {
	assert.Equal(t, "WS_SECRETS_MASTER_KEY", EnvSecretsKey)
	assert.Equal(t, "WS_SECRETS_MASTER_KEY_FILE", EnvSecretsKeyFile)
	assert.Equal(t, "WS_SECRETS_VAULT", EnvSecretsVault)
	assert.Equal(t, "WS_LOGGING_DIR", EnvLoggingDir)
	assert.Equal(t, "WS_LOGGING_MAIN_FILE", EnvLoggingFile)
	assert.Equal(t, "WS_SERVER_ROOT", EnvServerRoot)
	assert.Equal(t, "WS_FEATURES_DIR", EnvFeaturesDir)
	assert.Equal(t, "WS__INTERNAL_IPC_SOCKET", EnvIPCSocket)

	assert.Equal(t, "/etc/workspace/master.key", DefaultSecretsKeyPath)
	assert.Equal(t, "/var/log/workspace", DefaultLoggingDir)
	assert.Equal(t, "workspace.log", DefaultLoggingFile)
	assert.Equal(t, "/workspace", DefaultServerRoot)
	assert.Equal(t, "/features", DefaultFeaturesDir)
	assert.Equal(t, "/var/workspace/ipc.socket", DefaultIPCSocket)
	assert.Equal(t, "/var/lib/workspace/manifest.json", DefaultManifestPath)
}
