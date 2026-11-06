package features

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
)

const _storeManifest = `{
  "built": "2026-11-01T05:18:23Z",
  "packages": [
    {"name": "apache2-bin", "version": "2.4.68-1~deb13u1", "architecture": "amd64", "link": "pool/main/a/apache2/apache2-bin_2.4.68-1~deb13u1_amd64.deb", "type": "apt"},
    {"name": "bun", "version": "1.3.9", "architecture": "amd64", "link": "artifacts/bun/amd64.zip", "type": "artifact"},
    {"name": "bun", "version": "1.3.9", "architecture": "arm64", "link": "artifacts/bun/arm64.zip", "type": "artifact"},
    {"name": "composer", "version": null, "architecture": null, "link": "artifacts/composer/composer", "type": "artifact"},
    {"name": "gh", "version": "2.102.0", "architecture": "amd64", "link": "artifacts/gh/amd64.tar.gz", "type": "artifact"}
  ]
}`

func _store(t *testing.T, path, body string) string {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return srv.URL
}

func TestFetchStoreManifest(t *testing.T) {
	cases := []struct {
		name, path, base string
	}{
		{"Root", "/manifest.json", ""},
		{"TrailingSlash", "/manifest.json", "/"},
		{"Prefix", "/store/manifest.json", "/store"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			manifest, err := FetchStoreManifest(_store(t, c.path, _storeManifest) + c.base)
			assert.NilError(t, err)

			assert.Equal(t, manifest.Built, "2026-11-01T05:18:23Z")
			assert.DeepEqual(t, manifest.Artifacts(), []StorePackage{
				{Name: "bun", Version: "1.3.9", Type: "artifact"},
				{Name: "composer", Type: "artifact"},
				{Name: "gh", Version: "2.102.0", Type: "artifact"},
			})
		})
	}
}

func TestFetchStoreManifestErrors(t *testing.T) {
	cases := []struct {
		name, path, body, err string
	}{
		{"NotFound", "/artifacts/manifest.json", _storeManifest, "feature store returned status 404"},
		{"Invalid", "/manifest.json", "<html>", "failed to parse manifest"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := FetchStoreManifest(_store(t, c.path, c.body))
			assert.ErrorContains(t, err, c.err)
		})
	}
}
