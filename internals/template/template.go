package template

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

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

func lookup(name string) (Config, error) {
	config, exists := SupportedTemplates[name]
	if !exists {
		return Config{}, fmt.Errorf("template '%s' not found", name)
	}

	return config, nil
}

func GetTemplateNames() []string {
	return slices.Sorted(maps.Keys(SupportedTemplates))
}

func ApplyTemplate(name, targetPath string, force bool) error {
	config, err := lookup(name)
	if err != nil {
		return err
	}

	sourcePath := path.ResolveConfigPath(config.SourcePath)

	if !io.FileExists(sourcePath) {
		return fmt.Errorf("template source file not found: %s", sourcePath)
	}

	targetPath, err = filepath.Abs(targetPath)
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
	config, err := lookup(name)
	if err != nil {
		return "", err
	}

	sourcePath := path.ResolveConfigPath(config.SourcePath)
	if local {
		if sourcePath, err = path.GetCurrentWorkingDirectory(config.OutputName); err != nil {
			return "", err
		}
	}

	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template: %w", err)
	}

	return string(content), nil
}
