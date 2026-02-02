package config

const (
	EnvSecretsKey        = "WS_SECRETS_MASTER_KEY"
	EnvSecretsKeyFile    = "WS_SECRETS_MASTER_KEY_FILE"
	EnvSecretsVault      = "WS_SECRETS_VAULT"
	EnvLoggingDir        = "WS_LOGGING_DIR"
	EnvLoggingFile       = "WS_LOGGING_MAIN_FILE"
	EnvServerRoot        = "WS_SERVER_ROOT"
	EnvFeaturesDir       = "WS_FEATURES_DIR"
	EnvIPCSocket         = "WS__INTERNAL_IPC_SOCKET"
	EnvMetricsPort       = "WS_METRICS_PORT"
	EnvMetricsCollectors = "WS_METRICS_COLLECTORS"

	DefaultSecretsKeyPath = "/etc/workspace/master.key"
	DefaultEnvFilePath    = "~/.zshenv"
	DefaultLoggingDir     = "/var/log/workspace"
	DefaultLoggingFile    = "workspace.log"
	DefaultServerRoot     = "/workspace"
	DefaultFeaturesDir    = "/features"
	DefaultIPCSocket      = "/var/workspace/ipc.socket"
	DefaultManifestPath   = "/var/lib/workspace/manifest.json"
	DefaultStatePath      = "/var/lib/workspace/state"
	DefaultMetricsPort    = 9100
)
