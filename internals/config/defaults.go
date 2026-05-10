package config

const (
	EnvIPCSocket = "WS__INTERNAL_IPC_SOCKET"

	DefaultEnvReferencePath = "/etc/workspace/env.reference.yaml"
	DefaultIPCSocket        = "/var/workspace/ipc.socket"
	DefaultStatePath        = "/var/lib/workspace/state"
	DefaultEnvFilePath      = "~/.zshenv"
)

var DefaultManifestPath = "/var/lib/workspace/manifest.json"
