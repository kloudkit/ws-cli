package io

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestWriteSecureFile(t *testing.T) {
	cases := []struct {
		name, path, mode string
		existing         bool
		force            bool
		err              string
		perm, dirPerm    os.FileMode
		dir              string
	}{
		{name: "New", path: "a.txt", mode: "0o600", perm: 0o600},
		{name: "ExistsNoForce", path: "b.txt", mode: "0o600", existing: true, err: "exists, use --force to overwrite"},
		{name: "ExistsForce", path: "c.txt", mode: "0o600", perm: 0o600, existing: true, force: true},
		{name: "InvalidMode", path: "d.txt", mode: "999", err: "invalid file mode"},
		{name: "DefaultMode", path: "e.txt", perm: DefaultFileMode},
		{name: "Dir600", path: "d600/sub/f.txt", mode: "0o600", perm: 0o600, dir: "d600", dirPerm: 0o700},
		{name: "Dir644", path: "d644/sub/f.txt", mode: "0o644", perm: 0o644, dir: "d644", dirPerm: 0o744},
		{name: "Dir400", path: "d400/f.txt", mode: "0o400", perm: 0o400, dir: "d400", dirPerm: 0o700},
		{name: "Dir755", path: "d755/f.txt", mode: "0o755", perm: 0o755, dir: "d755", dirPerm: 0o755},
	}

	root := t.TempDir()

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(root, c.path)
			if c.existing {
				assert.NilError(t, os.WriteFile(path, []byte("original"), 0o644))
			}

			err := WriteSecureFile(path, []byte("content"), c.mode, c.force)

			data, _ := os.ReadFile(path)
			if c.err != "" {
				assert.ErrorContains(t, err, c.err)
				if c.existing {
					assert.Equal(t, string(data), "original")
				}
				return
			}

			assert.NilError(t, err)
			assert.Equal(t, string(data), "content")

			if info, err := os.Stat(path); c.perm != 0 {
				assert.NilError(t, err)
				assert.Equal(t, info.Mode().Perm(), c.perm)
			}

			if c.dir != "" {
				info, err := os.Stat(filepath.Join(root, c.dir))
				assert.NilError(t, err)
				assert.Equal(t, info.Mode().Perm(), c.dirPerm)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	source, dest := filepath.Join(dir, "source.txt"), filepath.Join(dir, "dest.txt")

	assert.Assert(t, !FileExists(source))
	assert.ErrorContains(t, CopyFile(source, dest), "failed to stat source file")
	assert.ErrorContains(t, CopyFile(dir, dest), "is not a regular file")

	assert.NilError(t, os.WriteFile(source, []byte("test content"), 0o644))
	assert.Assert(t, FileExists(source))
	assert.NilError(t, CopyFile(source, dest))

	data, err := os.ReadFile(dest)
	assert.NilError(t, err)
	assert.Equal(t, string(data), "test content")
}

func TestParseFileMode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    os.FileMode
		wantErr string
	}{
		{"EmptyStringReturnsDefault", "", DefaultFileMode, ""},
		{"OctalNotation0o600", "0o600", 0o600, ""},
		{"OctalNotation0O644", "0O644", 0o644, ""},
		{"DecimalNotation384", "384", 0o600, ""},
		{"DecimalNotation420", "420", 0o644, ""},
		{"InvalidFormat", "abc", 0, "invalid file mode"},
		{"ExceedsMaxMode", "0o1000", 0, "exceeds 0o777"},
		{"WhitespaceHandling", "  0o600  ", 0o600, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, err := ParseFileMode(tt.input)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			assert.NilError(t, err)
			assert.Equal(t, mode, tt.want)
		})
	}
}

func TestReadInputAndPassword(t *testing.T) {
	got, err := ReadInput("literal", nil)
	assert.NilError(t, err)
	assert.Equal(t, got, "literal")

	got, err = ReadInput("-", strings.NewReader("piped"))
	assert.NilError(t, err)
	assert.Equal(t, got, "piped")

	got, err = ReadPasswordFromReader(strings.NewReader("hunter2\n"))
	assert.NilError(t, err)
	assert.Equal(t, got, "hunter2\n")

	_, err = ReadPasswordFromReader(strings.NewReader("  \n"))
	assert.ErrorContains(t, err, "empty password provided")

	pipe := filepath.Join(t.TempDir(), "pw")
	assert.NilError(t, os.WriteFile(pipe, []byte("from-file"), 0o600))
	file, err := os.Open(pipe)
	assert.NilError(t, err)
	defer file.Close()

	got, err = ReadPasswordFromReader(file)
	assert.NilError(t, err)
	assert.Equal(t, got, "from-file")
}
