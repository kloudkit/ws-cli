package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetTemplate(t *testing.T) {
	config, exists := GetTemplate("markdownlint")
	if !exists {
		t.Error("markdownlint template should exist")
	}
	if config.SourcePath != ".config/markdownlint/config" {
		t.Errorf("expected source path '.config/markdownlint/config', got %s", config.SourcePath)
	}
	if config.OutputName != ".markdownlint.json" {
		t.Errorf("expected output name '.markdownlint.json', got %s", config.OutputName)
	}

	_, exists = GetTemplate("nonexistent")
	if exists {
		t.Error("nonexistent template should not exist")
	}
}

func TestGetTemplateNames(t *testing.T) {
	names := GetTemplateNames()
	expectedNames := []string{"ansible", "markdownlint", "ruff", "yamllint"}

	if len(names) != len(expectedNames) {
		t.Errorf("expected %d template names, got %d", len(expectedNames), len(names))
	}

	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	for _, expected := range expectedNames {
		if !nameSet[expected] {
			t.Errorf("expected template name '%s' not found", expected)
		}
	}
}

func TestApplyTemplate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "template_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, ".config", "markdownlint")
	err = os.MkdirAll(sourceDir, 0755)
	if err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	sourceFile := filepath.Join(sourceDir, "config")
	err = os.WriteFile(sourceFile, []byte(`{"line-length": false}`), 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	targetDir := filepath.Join(tempDir, "project")
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	err = ApplyTemplate("markdownlint", targetDir, false)
	if err != nil {
		t.Errorf("apply template failed: %v", err)
	}

	destFile := filepath.Join(targetDir, ".markdownlint.json")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Error("destination file was not created")
	}

	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Errorf("failed to read destination file: %v", err)
	}

	expected := `{"line-length": false}`
	if string(content) != expected {
		t.Errorf("expected content '%s', got '%s'", expected, string(content))
	}
}

func TestApplyTemplateWithForce(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "template_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, ".config", "ruff")
	err = os.MkdirAll(sourceDir, 0755)
	if err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	sourceFile := filepath.Join(sourceDir, "ruff.toml")
	err = os.WriteFile(sourceFile, []byte(`line-length = 88`), 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	targetDir := filepath.Join(tempDir, "project")
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	destFile := filepath.Join(targetDir, ".ruff.toml")
	err = os.WriteFile(destFile, []byte("existing content"), 0644)
	if err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	err = ApplyTemplate("ruff", targetDir, false)
	if err == nil {
		t.Error("expected error when file exists and force=false")
	}

	err = ApplyTemplate("ruff", targetDir, true)
	if err != nil {
		t.Errorf("apply template with force failed: %v", err)
	}

	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Errorf("failed to read destination file: %v", err)
	}

	expected := `line-length = 88`
	if string(content) != expected {
		t.Errorf("expected content '%s', got '%s'", expected, string(content))
	}
}

func TestShowTemplate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "template_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, ".config", "yamllint")
	err = os.MkdirAll(sourceDir, 0755)
	if err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	sourceFile := filepath.Join(sourceDir, "config")
	expectedContent := `extends: default
rules:
  line-length:
    max: 120`
	err = os.WriteFile(sourceFile, []byte(expectedContent), 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	content, err := ShowTemplate("yamllint", false)
	if err != nil {
		t.Errorf("show template failed: %v", err)
	}

	if content != expectedContent {
		t.Errorf("expected content '%s', got '%s'", expectedContent, content)
	}
}
