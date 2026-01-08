package template

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kloudkit/ws-cli/internals/io"
	"github.com/kloudkit/ws-cli/internals/path"
)

type Config struct {
	SourcePath string
	OutputName string
}

var SupportedTemplates = map[string]Config{
	"ansible": {
		SourcePath: "/etc/ansible/ansible.cfg",
		OutputName: "ansible.cfg",
	},
	"markdownlint": {
		SourcePath: ".config/markdownlint/config",
		OutputName: ".markdownlint.json",
	},
	"ruff": {
		SourcePath: ".config/ruff/ruff.toml",
		OutputName: ".ruff.toml",
	},
	"yamllint": {
		SourcePath: ".config/yamllint/config",
		OutputName: ".yamllint",
	},
}

func GetTemplate(name string) (Config, bool) {
	config, exists := SupportedTemplates[name]

	return config, exists
}

func GetTemplateNames() []string {
	names := make([]string, 0, len(SupportedTemplates))

	for name := range SupportedTemplates {
		names = append(names, name)
	}

	return names
}

func ApplyTemplate(name, targetPath string, force bool) error {
	config, exists := GetTemplate(name)
	if !exists {
		return fmt.Errorf("template '%s' not found", name)
	}

	sourcePath := path.ResolveConfigPath(config.SourcePath)

	if !io.FileExists(sourcePath) {
		return fmt.Errorf("template source file not found: %s", sourcePath)
	}

	targetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("invalid target path: %w", err)
	}

	destPath := path.AppendSegments(targetPath, config.OutputName)

	if !io.CanOverride(destPath, force) {
		return fmt.Errorf("file already exists: %s (use --force to overwrite)", destPath)
	}

	return io.CopyFile(sourcePath, destPath)
}

func ShowTemplate(name string, local bool) (string, error) {
	config, exists := GetTemplate(name)
	if !exists {
		return "", fmt.Errorf("template '%s' not found", name)
	}

	var sourcePath string
	var err error
	if local {
		sourcePath, err = path.GetCurrentWorkingDirectory(config.OutputName)
		if err != nil {
			return "", err
		}
	} else {
		sourcePath = path.ResolveConfigPath(config.SourcePath)
	}

	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template: %w", err)
	}

	return string(content), nil
}
