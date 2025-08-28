package features

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFeatureFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-feature.yaml")

	content := `---
- name: Install Test Feature
  gather_facts: false
  hosts: workspace
  vars:
    gpg: /etc/apt/keyrings/test.gpg
    repo: https://example.com/test
  tasks:
    - name: Test task
      ansible.builtin.apt:
        pkg:
          - test-package
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	feature, err := ParseFeatureFile(testFile)
	assert.NoError(t, err)

	assert.Equal(t, "test-feature", feature.Name)
	assert.Equal(t, "Install Test Feature", feature.Description)

	expectedVars := []string{"gpg", "repo"}
	assert.Len(t, feature.Vars, len(expectedVars))

	// Check that vars contain expected keys (order may vary due to map iteration)
	varMap := make(map[string]bool)
	for _, v := range feature.Vars {
		varMap[v] = true
	}

	for _, expected := range expectedVars {
		assert.True(t, varMap[expected], "Expected var '%s' not found", expected)
	}
}

func TestParseFeatureFileNoVars(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "no-vars.yaml")

	content := `---
- name: Install Feature Without Vars
  gather_facts: false
  hosts: workspace
  tasks:
    - name: Test task
      ansible.builtin.apt:
        pkg:
          - test-package
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	feature, err := ParseFeatureFile(testFile)
	assert.NoError(t, err)

	assert.Equal(t, "no-vars", feature.Name)
	assert.Empty(t, feature.Vars)
}

func TestParseFeatureFileInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.yaml")

	content := `invalid yaml content [[[`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := ParseFeatureFile(testFile)
	assert.Error(t, err)
}

func TestListFeatures(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test feature files
	testFiles := map[string]string{
		"feature1.yaml": `---
- name: First Feature
  gather_facts: false
  hosts: workspace
  vars:
    var1: value1
`,
		"feature2.yaml": `---
- name: Second Feature
  gather_facts: false
  hosts: workspace
`,
		"not-yaml.txt": "not a yaml file",
	}

	for filename, content := range testFiles {
		testFile := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	features, err := ListFeatures(tmpDir)
	assert.NoError(t, err)
	assert.Len(t, features, 2)

	// Check that we got both features
	featureNames := make(map[string]bool)
	for _, feature := range features {
		featureNames[feature.Name] = true
	}

	assert.True(t, featureNames["feature1"], "Expected 'feature1' not found")
	assert.True(t, featureNames["feature2"], "Expected 'feature2' not found")
}

func TestInfoFeature(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-feature.yaml")

	content := `---
- name: Install Test Feature
  gather_facts: false
  hosts: workspace
  vars:
    option1: value1
    option2: value2
  tasks:
    - name: Test task
      ansible.builtin.apt:
        pkg:
          - test-package
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	feature, err := InfoFeature(tmpDir, "test-feature")
	assert.NoError(t, err)

	assert.Equal(t, "test-feature", feature.Name)
	assert.Equal(t, "Install Test Feature", feature.Description)
	assert.Len(t, feature.Vars, 2)

	varMap := make(map[string]bool)
	for _, v := range feature.Vars {
		varMap[v] = true
	}

	assert.True(t, varMap["option1"], "Expected var 'option1' not found")
	assert.True(t, varMap["option2"], "Expected var 'option2' not found")
}

func TestInfoFeatureNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := InfoFeature(tmpDir, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "feature 'nonexistent' not found")
}
