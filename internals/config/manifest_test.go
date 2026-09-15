package config

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func _writeJSON(t *testing.T, name string, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	assert.NilError(t, os.WriteFile(path, []byte(body), 0o644))

	return path
}

func _withProductPath(t *testing.T, path string, fn func()) {
	t.Helper()
	original := DefaultProductJSONPath
	DefaultProductJSONPath = path
	defer func() { DefaultProductJSONPath = original }()
	fn()
}

func TestWorkspaceVersion_ReadsTheStampedManifest(t *testing.T) {
	path := _writeJSON(t, "manifest.json", `{"version":"v0.4.0-next"}`)

	_withManifestPath(t, path, func() {
		assert.Equal(t, "v0.4.0-next", WorkspaceVersion())
	})
}

func TestWorkspaceVersion_EmptyWhenManifestIsAbsent(t *testing.T) {
	_withManifestPath(t, filepath.Join(t.TempDir(), "absent.json"), func() {
		assert.Equal(t, "", WorkspaceVersion())
	})
}

func TestVSCodeVersion_ReadsTheServedProductJSON(t *testing.T) {
	path := _writeJSON(t, "product.json", `{"version":"1.129.0","nameLong":"Kloud Workspace"}`)

	_withProductPath(t, path, func() {
		version, err := VSCodeVersion()
		assert.NilError(t, err)
		assert.Equal(t, "1.129.0", version)
	})
}

func TestVSCodeVersion_ErrorsWhenProductJSONIsAbsent(t *testing.T) {
	_withProductPath(t, filepath.Join(t.TempDir(), "absent.json"), func() {
		_, err := VSCodeVersion()
		assert.ErrorContains(t, err, "failed to read product.json")
	})
}
