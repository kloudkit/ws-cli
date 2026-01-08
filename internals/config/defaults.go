package config

const (
	EnvSecretsKey     = "WS_SECRETS_MASTER_KEY"
	EnvSecretsKeyFile = "WS_SECRETS_MASTER_KEY_FILE"
	EnvLoggingDir     = "WS_LOGGING_DIR"
	EnvLoggingFile    = "WS_LOGGING_MAIN_FILE"
	EnvServerRoot     = "WS_SERVER_ROOT"
	EnvFeaturesDir    = "WS_FEATURES_DIR"
	EnvIPCSocket      = "WS__INTERNAL_IPC_SOCKET"

	DefaultSecretsKeyPath = "/etc/workspace/master.key"
	DefaultLoggingDir     = "/var/log/workspace"
	DefaultLoggingFile    = "workspace.log"
	DefaultServerRoot     = "/workspace"
	DefaultFeaturesDir    = "/features"
	DefaultIPCSocket      = "/var/workspace/ipc.socket"
	DefaultManifestPath   = "/var/lib/workspace/manifest.json"
)
