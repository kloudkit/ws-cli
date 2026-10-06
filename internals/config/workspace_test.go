package config

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func _withManifestPath(t *testing.T, path string, fn func()) {
	t.Helper()
	original := DefaultManifestPath
	DefaultManifestPath = path
	defer func() { DefaultManifestPath = original }()
	fn()
}

func TestBootstrap_RequiresManifestFile(t *testing.T) {
	_installFixture(t, sampleYAML)
	f, err := os.CreateTemp(t.TempDir(), "manifest*.json")
	assert.NilError(t, err)
	f.Close()

	cases := []struct {
		name, path string
		ok         bool
	}{
		{"FileExists", f.Name(), true},
		{"FileAbsent", filepath.Join(t.TempDir(), "nonexistent.json"), false},
		{"PathIsDirectory", t.TempDir(), false},
		{"EmptyPath", "", false},
	}
	for _, c := range cases {
		_withManifestPath(t, c.path, func() {
			err := Bootstrap()
			if c.ok {
				assert.NilError(t, err, c.name)
			} else {
				assert.ErrorContains(t, err, "Workspace", c.name)
			}
		})
	}
}
