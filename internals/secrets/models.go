package secrets

import (
	"fmt"
	"os"
	filepath "path/filepath"
	"regexp"
	"strings"

	"github.com/kloudkit/ws-cli/internals/path"
)

const (
	EnvMasterKey      = "WS_SECRETS_MASTER_KEY"
	EnvMasterKeyFile  = "WS_SECRETS_MASTER_KEY_FILE"
	DefaultMasterPath = "/etc/workspace/master.key"

	FileModeKubeconfig os.FileMode = 0600
	FileModeSSH        os.FileMode = 0600
	FileModePassword   os.FileMode = 0600
	FileModeConfig     os.FileMode = 0644
	FileModeDefault    os.FileMode = 0600
)

var (
	allowedPaths = []string{
		"/home/dev/.kube/",
		"/home/dev/.ssh/",
		"/etc/secrets/",
	}

	envVarNameRegex = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

	typeFileModes = map[string]os.FileMode{
		"kubeconfig": FileModeKubeconfig,
		"ssh":        FileModeSSH,
		"password":   FileModePassword,
		"config":     FileModeConfig,
	}
)

type Secret struct {
	Type        string `yaml:"type"`
	Value       string `yaml:"value"`
	Destination string `yaml:"destination"`
	Force       bool   `yaml:"force,omitempty"`
}

type Vault struct {
	Secrets []Secret `yaml:"secrets"`
}

func (s *Secret) IsEnvDestination() bool {
	return envVarNameRegex.MatchString(s.Destination)
}

func (s *Secret) ExpandedDestination() (string, error) {
	if s.IsEnvDestination() {
		return s.Destination, nil
	}

	dest := s.Destination

	if strings.HasPrefix(dest, "~/") {
		homeDir := path.GetHomeDirectory()
		dest = filepath.Join(homeDir, dest[2:])
	}

	dest = os.ExpandEnv(dest)

	absPath, err := filepath.Abs(dest)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	return filepath.Clean(absPath), nil
}

func (s *Secret) ValidateDestination() error {
	if s.Destination == "" {
		return fmt.Errorf("destination cannot be empty")
	}

	if s.IsEnvDestination() {
		return nil
	}

	expanded, err := s.ExpandedDestination()
	if err != nil {
		return err
	}

	for _, allowed := range allowedPaths {
		if strings.HasPrefix(expanded, allowed) {
			return nil
		}
	}

	return fmt.Errorf("path %s is not in allowed directories: %v", expanded, allowedPaths)
}

func (s *Secret) FileMode() os.FileMode {
	if mode, ok := typeFileModes[s.Type]; ok {
		return mode
	}

	return FileModeDefault
}

func (s *Secret) ReadPlaintextValue() ([]byte, error) {
	if err := s.ValidateDestination(); err != nil {
		return nil, err
	}

	if s.IsEnvDestination() {
		value := os.Getenv(s.Destination)
		if value == "" {
			return nil, fmt.Errorf("environment variable %s is not set", s.Destination)
		}
		return []byte(value), nil
	}

	expanded, err := s.ExpandedDestination()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(expanded)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", expanded, err)
	}

	return data, nil
}
