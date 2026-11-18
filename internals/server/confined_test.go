package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func confinedSite(t *testing.T) http.Handler {
	t.Helper()

	outside := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644))

	root := t.TempDir()
	for name, content := range map[string]string{
		"index.html":        "home",
		".env":              "TOKEN=x",
		"docs/page.html":    "page",
		"empty/readme.txt":  "readme",
		".git/config":       "[core]",
		"assets/.hidden.js": "hidden",
	} {
		path := filepath.Join(root, name)
		assert.NilError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		assert.NilError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	assert.NilError(t, os.WriteFile(filepath.Join(root, "docs", "index.html"), []byte("docs"), 0o644))
	assert.NilError(t, os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "leak.txt")))
	assert.NilError(t, os.Symlink(outside, filepath.Join(root, "out")))
	assert.NilError(t, os.Symlink(filepath.Join("docs", "page.html"), filepath.Join(root, "inside.html")))
	assert.NilError(t, os.Symlink(".env", filepath.Join(root, "cfg")))
	assert.NilError(t, os.Symlink(".git", filepath.Join(root, "repo")))

	handler, err := Confined(root)
	assert.NilError(t, err)

	return handler
}

func TestConfinedServesFilesInsideTheRoot(t *testing.T) {
	handler := confinedSite(t)

	for path, body := range map[string]string{"/": "home", "/docs/page.html": "page", "/docs/": "docs", "/inside.html": "page"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

		assert.Equal(t, rec.Code, http.StatusOK, path)
		assert.Equal(t, rec.Body.String(), body, path)
	}
}

func TestConfinedAnswers404ForDotfilesEscapesAndListings(t *testing.T) {
	handler := confinedSite(t)

	for _, path := range []string{"/.env", "/.git/config", "/assets/.hidden.js", "/cfg", "/repo/config", "/leak.txt", "/out/secret.txt", "/empty/", "/empty", "/missing"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

		assert.Equal(t, rec.Code, http.StatusNotFound, path)
	}
}
