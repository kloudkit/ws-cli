package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/env"
	internalIO "github.com/kloudkit/ws-cli/internals/io"
	"github.com/kloudkit/ws-cli/internals/path"
	"gopkg.in/yaml.v3"
)

type VaultSecret struct {
	Type        string `yaml:"type,omitempty"`
	Encrypted   string `yaml:"encrypted"`
	Destination string `yaml:"destination"`
	Mode        string `yaml:"mode,omitempty"`
	Force       bool   `yaml:"force,omitempty"`
}

type Vault struct {
	Secrets map[string]VaultSecret `yaml:"secrets"`
}

const (
	TypeGeneric          = "generic"
	TypeSSH              = "ssh"
	TypeEnv              = "env"
	TypeKubeconfig       = "kubeconfig"
	TypeDockerConfigJSON = "dockerconfigjson"
)

type SecretTypeConfig struct {
	DefaultMode      string
	DefaultDirectory string
}

var SecretTypeConfigs = map[string]SecretTypeConfig{
	TypeGeneric: {
		DefaultMode:      "0o600",
		DefaultDirectory: "",
	},
	TypeSSH: {
		DefaultMode:      "0o600",
		DefaultDirectory: "~/.ssh",
	},
	TypeEnv: {
		DefaultMode:      "0o644",
		DefaultDirectory: "",
	},
	TypeKubeconfig: {
		DefaultMode:      "0o600",
		DefaultDirectory: "~/.kube",
	},
	TypeDockerConfigJSON: {
		DefaultMode:      "0o600",
		DefaultDirectory: "~/.docker",
	},
}

func LoadVault(path string) (*Vault, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read vault file %q: %w", path, err)
	}

	var vault Vault
	if err := yaml.Unmarshal(data, &vault); err != nil {
		return nil, fmt.Errorf("failed to unmarshal vault yaml: %w", err)
	}

	if vault.Secrets == nil {
		vault.Secrets = make(map[string]VaultSecret)
	}

	for name, secret := range vault.Secrets {
		if secret.Type == "" {
			secret.Type = TypeGeneric
		}

		if secret.Mode == "" {
			if config, ok := SecretTypeConfigs[secret.Type]; ok {
				secret.Mode = config.DefaultMode
			}
		}

		resolvedDest, err := ResolveDestination(secret)
		if err != nil {
			return nil, fmt.Errorf("secret %q: %w", name, err)
		}
		secret.Destination = resolvedDest

		vault.Secrets[name] = secret
	}

	return &vault, nil
}

func ResolveDestination(secret VaultSecret) (string, error) {
	if secret.Type == TypeEnv {
		return secret.Destination, nil
	}

	config, ok := SecretTypeConfigs[secret.Type]
	if !ok {
		return "", fmt.Errorf("unknown type %q", secret.Type)
	}

	if filepath.IsAbs(secret.Destination) || strings.HasPrefix(secret.Destination, "~") {
		resolved, err := path.Expand(secret.Destination)
		if err != nil {
			return "", fmt.Errorf("failed to expand path: %w", err)
		}
		return resolved, nil
	}

	if config.DefaultDirectory == "" {
		return "", fmt.Errorf("type %q requires an absolute path", secret.Type)
	}

	fullPath := filepath.Join(config.DefaultDirectory, secret.Destination)
	resolved, err := path.Expand(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to expand path: %w", err)
	}
	return resolved, nil
}

func ResolveVaultPath(inputFlag string) (string, error) {
	if inputFlag != "" {
		return inputFlag, nil
	}

	if vaultPath := env.String(config.EnvSecretsVault); vaultPath != "" {
		return vaultPath, nil
	}

	return "", fmt.Errorf("vault file not specified (use --input or %s)", config.EnvSecretsVault)
}

func ValidateSecret(name string, secret VaultSecret) error {
	if secret.Encrypted == "" {
		return fmt.Errorf("secret %q: encrypted value is required", name)
	}

	if secret.Destination == "" {
		return fmt.Errorf("secret %q: destination is required", name)
	}

	validTypes := []string{
		TypeGeneric,
		TypeSSH,
		TypeEnv,
		TypeKubeconfig,
		TypeDockerConfigJSON,
	}

	if !slices.Contains(validTypes, secret.Type) {
		return fmt.Errorf("secret %q: invalid type %q", name, secret.Type)
	}

	if secret.Type == TypeEnv {
		if !env.IsValidName(secret.Destination) {
			return fmt.Errorf("secret %q: invalid environment variable name %q (must start with letter/underscore and contain only alphanumerics and underscores)", name, secret.Destination)
		}
	} else if !filepath.IsAbs(secret.Destination) {
		return fmt.Errorf("secret %q: invalid destination path", name)
	}

	return nil
}

func GetSecretKeys(vault *Vault, requestedKeys []string) []string {
	if len(requestedKeys) > 0 {
		return requestedKeys
	}

	keys := make([]string, 0, len(vault.Secrets))
	for key := range vault.Secrets {
		keys = append(keys, key)
	}

	return keys
}

type ProcessOptions struct {
	MasterKey    []byte
	Keys         []string
	Stdout       bool
	Raw          bool
	Force        bool
	ModeOverride string
}

func ProcessVault(vault *Vault, opts ProcessOptions) (map[string]string, error) {
	results := make(map[string]string)
	keys := GetSecretKeys(vault, opts.Keys)

	for _, key := range keys {
		secret, exists := vault.Secrets[key]
		if !exists {
			return nil, fmt.Errorf("secret %q not found in vault", key)
		}

		if err := ValidateSecret(key, secret); err != nil {
			return nil, err
		}

		encryptedValue := NormalizeEncrypted(secret.Encrypted)

		decrypted, err := Decrypt(encryptedValue, opts.MasterKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt secret %q: %w", key, err)
		}

		if opts.Stdout {
			results[key] = string(decrypted)
			continue
		}

		mode := secret.Mode
		if opts.ModeOverride != "" {
			mode = opts.ModeOverride
		}

		if secret.Type == TypeEnv {
			if err := ProcessEnvSecret(secret.Destination, decrypted, opts.Force); err != nil {
				return nil, fmt.Errorf("failed to process env secret %q: %w", key, err)
			}
			results[key] = fmt.Sprintf("env:%s", secret.Destination)
		} else {
			if err := internalIO.WriteSecureFile(secret.Destination, decrypted, mode, opts.Force); err != nil {
				return nil, fmt.Errorf("failed to write secret %q: %w", key, err)
			}
			results[key] = secret.Destination
		}
	}

	return results, nil
}

func findAndReplaceEnvVar(lines []string, envVarName, value string, force bool) ([]string, error) {
	exportLine := fmt.Sprintf("export %s=%q", envVarName, value)
	found := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "export "+envVarName+"=") ||
			strings.HasPrefix(trimmed, envVarName+"=") {
			if !force {
				return nil, fmt.Errorf("environment variable %q already exists, use --force to overwrite", envVarName)
			}
			lines[i] = exportLine
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, exportLine)
	}

	return lines, nil
}

func ProcessEnvSecret(envVarName string, value []byte, force bool) error {
	envFilePath, err := path.Expand(config.DefaultEnvFilePath)
	if err != nil {
		return err
	}

	var existingContent []byte
	if internalIO.FileExists(envFilePath) {
		data, err := os.ReadFile(envFilePath)
		if err != nil {
			return fmt.Errorf("failed to read env file: %w", err)
		}
		existingContent = data
	}

	lines := strings.Split(string(existingContent), "\n")

	lines, err = findAndReplaceEnvVar(lines, envVarName, string(value), force)
	if err != nil {
		return fmt.Errorf("%w in %s", err, envFilePath)
	}

	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	if err := os.WriteFile(envFilePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}

	return nil
}

func FormatSecretForStdout(key string, value string, raw bool) string {
	if raw {
		return value
	}

	return fmt.Sprintf("[%s]\n%s\n", key, value)
}
