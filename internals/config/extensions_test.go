package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func _fakeCode(t *testing.T, script string) {
	t.Helper()

	dir := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(dir, "code"), []byte("#!/bin/sh\n"+script+"\n"), 0o755))
	t.Setenv("PATH", dir)
}

func TestGetExtensions(t *testing.T) {
	cases := []struct {
		name   string
		script string
		want   []Extension
		err    string
	}{
		{"Parses", "printf 'a.b@1.0.0\\nc.d@2.1\\nbroken\\n'", []Extension{{"a.b", "1.0.0"}, {"c.d", "2.1"}}, ""},
		{"Empty", "true", nil, ""},
		{"Fails", "exit 3", nil, "`code --list-extensions` failed: exit status 3"},
		{"Hangs", "/bin/sleep 5", nil, "`code --list-extensions` did not answer within 100ms"},
	}

	original := extensionsTimeout
	extensionsTimeout = 100 * time.Millisecond
	defer func() { extensionsTimeout = original }()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_fakeCode(t, tc.script)

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
