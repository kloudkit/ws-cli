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

	assert.DeepEqual(t, []string{"gpg", "repo"}, feature.Vars)
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

func TestParseFeatureFileEmptyArray(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.yaml")

	err := os.WriteFile(testFile, []byte("---\n[]\n"), 0644)
	assert.NilError(t, err)

	_, err = ParseFeatureFile(testFile)
	assert.ErrorContains(t, err, "no playbook tasks found")
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

	result, err := ListFeatures([]string{tmpDir})
	assert.NilError(t, err)
	assert.Equal(t, 2, len(result.Features))
	assert.Equal(t, 0, len(result.Warnings))

	featureNames := make(map[string]bool)
	for _, feature := range result.Features {
		featureNames[feature.Name] = true
	}

	assert.Assert(t, featureNames["feature1"])
	assert.Assert(t, featureNames["feature2"])
}

func TestListFeaturesWithWarnings(t *testing.T) {
	tmpDir := t.TempDir()

	testFiles := map[string]string{
		"good.yaml": `---
- name: Good Feature
  gather_facts: false
  hosts: workspace
`,
		"bad.yaml": "invalid yaml [[[",
	}

	for filename, content := range testFiles {
		err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
		assert.NilError(t, err)
	}

	result, err := ListFeatures([]string{tmpDir})
	assert.NilError(t, err)
	assert.Equal(t, 1, len(result.Features))
	assert.Equal(t, "good", result.Features[0].Name)
	assert.Equal(t, 1, len(result.Warnings))
	assert.Assert(t, len(result.Warnings[0]) > 0)
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

	feature, err := InfoFeature([]string{tmpDir}, "test-feature")
	assert.NilError(t, err)

	assert.Equal(t, "test-feature", feature.Name)
	assert.Equal(t, "Install Test Feature", feature.Description)
	assert.DeepEqual(t, []string{"option1", "option2"}, feature.Vars)
}

func TestInfoFeatureNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := InfoFeature([]string{tmpDir}, "nonexistent")
	assert.ErrorContains(t, err, "feature 'nonexistent' not found")
}

func _writeFeature(t *testing.T, dir, name, desc string) {
	t.Helper()
	content := "---\n- name: " + desc + "\n  gather_facts: false\n  hosts: workspace\n"
	assert.NilError(t, os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(content), 0644))
}

func _writeRaw(t *testing.T, dir, name, content string) {
	t.Helper()
	assert.NilError(t, os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(content), 0644))
}

func TestListFeaturesUserOverridesSystem(t *testing.T) {
	system, user := t.TempDir(), t.TempDir()
	_writeFeature(t, system, "git", "System Git")
	_writeFeature(t, user, "git", "User Git")

	result, err := ListFeatures([]string{system, user})
	assert.NilError(t, err)
	assert.Equal(t, 1, len(result.Features))
	assert.Equal(t, "git", result.Features[0].Name)
	assert.Equal(t, "User Git", result.Features[0].Description)
	assert.Equal(t, SourceOverride, result.Features[0].Source)
}

func TestListFeaturesUnionDistinctNames(t *testing.T) {
	system, user := t.TempDir(), t.TempDir()
	_writeFeature(t, system, "only-system", "System Only")
	_writeFeature(t, user, "only-user", "User Only")

	result, err := ListFeatures([]string{system, user})
	assert.NilError(t, err)
	assert.Equal(t, 2, len(result.Features))
}

func TestListFeaturesUserDirAbsent(t *testing.T) {
	system := t.TempDir()
	_writeFeature(t, system, "git", "System Git")
	missing := filepath.Join(t.TempDir(), "nope")

	result, err := ListFeatures([]string{system, missing})
	assert.NilError(t, err)
	assert.Equal(t, 1, len(result.Features))
	assert.Equal(t, SourceSystem, result.Features[0].Source)
}

func TestListFeaturesSystemDirAbsent(t *testing.T) {
	user := t.TempDir()
	_writeFeature(t, user, "git", "User Git")
	missing := filepath.Join(t.TempDir(), "nope")

	result, err := ListFeatures([]string{missing, user})
	assert.NilError(t, err)
	assert.Equal(t, 1, len(result.Features))
	assert.Equal(t, SourceUser, result.Features[0].Source)
}

func TestListFeaturesUserDirEmpty(t *testing.T) {
	system, user := t.TempDir(), t.TempDir()
	_writeFeature(t, system, "git", "System Git")

	result, err := ListFeatures([]string{system, user})
	assert.NilError(t, err)
	assert.Equal(t, 1, len(result.Features))
	assert.Equal(t, 0, len(result.Warnings))
}

func TestListFeaturesMalformedUserNoTwin(t *testing.T) {
	system, user := t.TempDir(), t.TempDir()
	_writeFeature(t, system, "good", "Good Feature")
	_writeRaw(t, user, "bad", "invalid yaml [[[")

	result, err := ListFeatures([]string{system, user})
	assert.NilError(t, err)
	assert.Equal(t, 1, len(result.Features))
	assert.Equal(t, "good", result.Features[0].Name)
	assert.Equal(t, 1, len(result.Warnings))
}

func TestListFeaturesMalformedUserShadowsSystem_NoFallback(t *testing.T) {
	system, user := t.TempDir(), t.TempDir()
	_writeFeature(t, system, "git", "System Git")
	_writeRaw(t, user, "git", "invalid yaml [[[")

	result, err := ListFeatures([]string{system, user})
	assert.NilError(t, err)
	assert.Equal(t, 0, len(result.Features))
	assert.Equal(t, 1, len(result.Warnings))
}

func TestListFeaturesMarksSource(t *testing.T) {
	system, user := t.TempDir(), t.TempDir()
	_writeFeature(t, system, "sys-only", "System Only")
	_writeFeature(t, system, "shared", "System Shared")
	_writeFeature(t, user, "shared", "User Shared")
	_writeFeature(t, user, "usr-only", "User Only")

	result, err := ListFeatures([]string{system, user})
	assert.NilError(t, err)

	byName := map[string]*Feature{}
	for _, f := range result.Features {
		byName[f.Name] = f
	}
	assert.Equal(t, SourceSystem, byName["sys-only"].Source)
	assert.Equal(t, SourceOverride, byName["shared"].Source)
	assert.Equal(t, SourceUser, byName["usr-only"].Source)
}

func TestInfoFeatureUserWins(t *testing.T) {
	system, user := t.TempDir(), t.TempDir()
	_writeFeature(t, system, "git", "System Git")
	_writeFeature(t, user, "git", "User Git")

	feature, err := InfoFeature([]string{system, user}, "git")
	assert.NilError(t, err)
	assert.Equal(t, "User Git", feature.Description)
	assert.Equal(t, SourceOverride, feature.Source)
}

func TestInfoFeatureSystemFallback(t *testing.T) {
	system, user := t.TempDir(), t.TempDir()
	_writeFeature(t, system, "git", "System Git")

	feature, err := InfoFeature([]string{system, user}, "git")
	assert.NilError(t, err)
	assert.Equal(t, "System Git", feature.Description)
	assert.Equal(t, SourceSystem, feature.Source)
}

func TestInfoFeatureMalformedUserShadow_Errors(t *testing.T) {
	system, user := t.TempDir(), t.TempDir()
	_writeFeature(t, system, "git", "System Git")
	_writeRaw(t, user, "git", "invalid yaml [[[")

	_, err := InfoFeature([]string{system, user}, "git")
	assert.Assert(t, err != nil)
}

func TestInfoFeatureNotFoundInEither(t *testing.T) {
	system, user := t.TempDir(), t.TempDir()

	_, err := InfoFeature([]string{system, user}, "nope")
	assert.ErrorContains(t, err, "feature 'nope' not found")
}

func TestResolveFeaturePathUserWins(t *testing.T) {
	system, user := t.TempDir(), t.TempDir()
	_writeFeature(t, system, "git", "System Git")
	_writeFeature(t, user, "git", "User Git")

	path, err := ResolveFeaturePath([]string{system, user}, "git")
	assert.NilError(t, err)
	assert.Equal(t, filepath.Join(user, "git.yaml"), path)
}

func TestResolveFeaturePathNotFound(t *testing.T) {
	system := t.TempDir()

	_, err := ResolveFeaturePath([]string{system}, "nope")
	assert.ErrorContains(t, err, "feature 'nope' not found")
}
