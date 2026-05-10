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

func TestIsWorkspace_FileExists(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "manifest*.json")
	assert.NilError(t, err)
	f.Close()
	_withManifestPath(t, f.Name(), func() {
		assert.Equal(t, true, IsWorkspace())
	})
}

func TestIsWorkspace_FileAbsent(t *testing.T) {
	_withManifestPath(t, filepath.Join(t.TempDir(), "nonexistent.json"), func() {
		assert.Equal(t, false, IsWorkspace())
	})
}

func TestIsWorkspace_PathIsDirectory(t *testing.T) {
	_withManifestPath(t, t.TempDir(), func() {
		assert.Equal(t, false, IsWorkspace())
	})
}

func TestIsWorkspace_EmptyPath(t *testing.T) {
	_withManifestPath(t, "", func() {
		assert.Equal(t, false, IsWorkspace())
	})
}

func TestRequireWorkspace_FileExists(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "manifest*.json")
	assert.NilError(t, err)
	f.Close()
	_withManifestPath(t, f.Name(), func() {
		assert.NilError(t, RequireWorkspace())
	})
}

func TestRequireWorkspace_FileAbsent(t *testing.T) {
	_withManifestPath(t, filepath.Join(t.TempDir(), "nonexistent.json"), func() {
		err := RequireWorkspace()
		assert.ErrorContains(t, err, "Workspace")
	})
}

func TestRequireWorkspace_PathIsDirectory(t *testing.T) {
	_withManifestPath(t, t.TempDir(), func() {
		err := RequireWorkspace()
		assert.Assert(t, err != nil)
	})
}
