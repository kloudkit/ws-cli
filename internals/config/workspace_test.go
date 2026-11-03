package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func _override(t *testing.T, target *string, value string) {
	t.Helper()
	original := *target
	*target = value
	t.Cleanup(func() { *target = original })
}

func _writeFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "file.json")
	assert.NilError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

func TestBootstrap_RequiresManifestFile(t *testing.T) {
	_installFixture(t, sampleYAML)
	cases := map[string]string{
		"FileExists":      _writeFile(t, ""),
		"FileAbsent":      filepath.Join(t.TempDir(), "absent.json"),
		"PathIsDirectory": t.TempDir(),
		"EmptyPath":       "",
	}
	for name, path := range cases {
		_override(t, &DefaultManifestPath, path)
		err := Bootstrap()
		if name == "FileExists" {
			assert.NilError(t, err, name)
		} else {
			assert.ErrorContains(t, err, "Workspace", name)
		}
	}
}

func TestVersions(t *testing.T) {
	_override(t, &DefaultManifestPath, _writeFile(t, `{"version":"v0.4.0-next"}`))
	assert.Equal(t, "v0.4.0-next", WorkspaceVersion())

	_override(t, &DefaultManifestPath, filepath.Join(t.TempDir(), "absent.json"))
	assert.Equal(t, "", WorkspaceVersion())

	_override(t, &DefaultProductJSONPath, _writeFile(t, `{"version":"1.129.0","nameLong":"Kloud Workspace"}`))
	version, err := VSCodeVersion()
	assert.NilError(t, err)
	assert.Equal(t, "1.129.0", version)

	_override(t, &DefaultProductJSONPath, filepath.Join(t.TempDir(), "absent.json"))
	_, err = VSCodeVersion()
	assert.ErrorContains(t, err, "failed to read product.json")
}

func TestGetExtensions(t *testing.T) {
	cases := []struct {
		name, script string
		want         []Extension
		err          string
	}{
		{"Parses", "printf 'a.b@1.0.0\\nc.d@2.1\\nbroken\\ne@f@g\\n'", []Extension{{"a.b", "1.0.0"}, {"c.d", "2.1"}}, ""},
		{"Empty", "true", nil, ""},
		{"Fails", "exit 3", nil, "`code --list-extensions` failed: exit status 3"},
		{"Hangs", "/bin/sleep 5", nil, "`code --list-extensions` did not answer within 100ms"},
	}

	original := extensionsTimeout
	extensionsTimeout = 100 * time.Millisecond
	t.Cleanup(func() { extensionsTimeout = original })

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			assert.NilError(t, os.WriteFile(filepath.Join(dir, "code"), []byte("#!/bin/sh\n"+tc.script+"\n"), 0o755))
			t.Setenv("PATH", dir)

			started := time.Now()
			got, err := GetExtensions()
			assert.Assert(t, time.Since(started) < 2*time.Second)

			if tc.err != "" {
				assert.Error(t, err, tc.err)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, got, tc.want)
		})
	}
}
