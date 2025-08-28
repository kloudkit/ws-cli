package template

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

	var sourcePath string
	if strings.HasPrefix(config.SourcePath, "/") {
		sourcePath = config.SourcePath
	} else {
		sourcePath = path.GetHomeDirectory(config.SourcePath)
	}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("template source file not found: %s", sourcePath)
	}

	targetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("invalid target path: %w", err)
	}

	destPath := path.AppendSegments(targetPath, config.OutputName)

	if !path.CanOverride(destPath, force) {
		return fmt.Errorf("file already exists: %s (use --force to overwrite)", destPath)
	}

	return copyFile(sourcePath, destPath)
}

func ShowTemplate(name string, local bool) (string, error) {
	config, exists := GetTemplate(name)
	if !exists {
		return "", fmt.Errorf("template '%s' not found", name)
	}

	var sourcePath string
	if local {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		sourcePath = path.AppendSegments(cwd, config.OutputName)
	} else {
		if strings.HasPrefix(config.SourcePath, "/") {
			sourcePath = config.SourcePath
		} else {
			sourcePath = path.GetHomeDirectory(config.SourcePath)
		}
	}

	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template: %w", err)
	}

	return string(content), nil
}

func copyFile(source, dest string) error {
	stats, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	if !stats.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", source)
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
