package features

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
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

	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NilError(t, err)

	feature, err := ParseFeatureFile(testFile)
	assert.NilError(t, err)

	assert.Equal(t, "test-feature", feature.Name)
	assert.Equal(t, "Install Test Feature", feature.Description)

	expectedVars := []string{"gpg", "repo"}
	assert.Equal(t, len(expectedVars), len(feature.Vars))

	varMap := make(map[string]bool)
	for _, v := range feature.Vars {
		varMap[v] = true
	}

	for _, expected := range expectedVars {
		assert.Assert(t, varMap[expected])
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

	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NilError(t, err)

	feature, err := ParseFeatureFile(testFile)
	assert.NilError(t, err)

	assert.Equal(t, "no-vars", feature.Name)
	assert.Equal(t, 0, len(feature.Vars))
}

func TestParseFeatureFileInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.yaml")

	content := `invalid yaml content [[[`

	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NilError(t, err)

	_, err = ParseFeatureFile(testFile)
	assert.Assert(t, err != nil)
}

func TestListFeatures(t *testing.T) {
	tmpDir := t.TempDir()

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
		err := os.WriteFile(testFile, []byte(content), 0644)
		assert.NilError(t, err)
	}

	features, err := ListFeatures(tmpDir)
	assert.NilError(t, err)
	assert.Equal(t, 2, len(features))

	featureNames := make(map[string]bool)
	for _, feature := range features {
		featureNames[feature.Name] = true
	}

	assert.Assert(t, featureNames["feature1"])
	assert.Assert(t, featureNames["feature2"])
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

	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NilError(t, err)

	feature, err := InfoFeature(tmpDir, "test-feature")
	assert.NilError(t, err)

	assert.Equal(t, "test-feature", feature.Name)
	assert.Equal(t, "Install Test Feature", feature.Description)
	assert.Equal(t, 2, len(feature.Vars))

	varMap := make(map[string]bool)
	for _, v := range feature.Vars {
		varMap[v] = true
	}

	assert.Assert(t, varMap["option1"])
	assert.Assert(t, varMap["option2"])
}

func TestInfoFeatureNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := InfoFeature(tmpDir, "nonexistent")
	assert.ErrorContains(t, err, "feature 'nonexistent' not found")
}
