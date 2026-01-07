package secrets

import (
	"bufio"
	"fmt"
	"os"
	filepath "path/filepath"
	"strings"

	"github.com/kloudkit/ws-cli/internals/path"
)

const (
	EnvFile = ".zshenv"
)

type WriteOptions struct {
	Force  bool
	DryRun bool
}

func WriteSecret(secret *Secret, decryptedValue []byte, opts WriteOptions) error {
	if err := secret.ValidateDestination(); err != nil {
		return err
	}

	expanded, err := secret.ExpandedDestination()
	if err != nil {
		return err
	}

	if secret.IsEnvDestination() {
		return writeEnvVar(expanded, string(decryptedValue), opts)
	}

	return writeFile(expanded, decryptedValue, secret.FileMode(), opts)
}

func writeFile(filePath string, content []byte, mode os.FileMode, opts WriteOptions) error {
	if opts.DryRun {
		fmt.Printf("[DRY-RUN] Would write to file: %s (mode: %04o)\n", filePath, mode)
		return nil
	}

	if !opts.Force && path.FileExists(filePath) {
		return fmt.Errorf("file %s already exists, use --force to overwrite", filePath)
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(filePath, content, mode); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

func writeEnvVar(varName, value string, opts WriteOptions) error {
	envFilePath := path.GetHomeDirectory(EnvFile)

	if opts.DryRun {
		fmt.Printf("[DRY-RUN] Would append to %s: export %s=<value>\n", envFilePath, varName)
		return nil
	}

	exists, err := envVarExists(envFilePath, varName)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("environment variable %s already exists in %s, skipping", varName, envFilePath)
	}

	file, err := os.OpenFile(envFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", envFilePath, err)
	}
	defer file.Close()

	line := fmt.Sprintf("export %s=%s\n", varName, value)
	if _, err := file.WriteString(line); err != nil {
		return fmt.Errorf("failed to write to %s: %w", envFilePath, err)
	}

	return nil
}

func envVarExists(envFilePath, varName string) (bool, error) {
	if !path.FileExists(envFilePath) {
		return false, nil
	}

	file, err := os.Open(envFilePath)
	if err != nil {
		return false, fmt.Errorf("failed to open %s: %w", envFilePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	prefix := fmt.Sprintf("export %s=", varName)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, prefix) {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("error reading %s: %w", envFilePath, err)
	}

	return false, nil
}
