package seed

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func lineinfileManifest(dest, content string) string {
	return fmt.Sprintf("seeds:\n  %s:\n    op: lineinfile\n    content: %s\n", dest, content)
}

func TestEnsureLine(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		body     string
		want     string
		err      string
	}{
		{
			name:     "AppendsWhenAbsent",
			existing: "A=1\n",
			body:     "FOO=bar\n",
			want:     "A=1\nFOO=bar\n",
		},
		{
			name:     "ReplacesInPlaceWhenPresent",
			existing: "FOO=old\n",
			body:     "FOO=new\n",
			want:     "FOO=new\n",
		},
		{
			name:     "PreservesSurroundingLines",
			existing: "A=1\nFOO=old\nB=2\n",
			body:     "FOO=new\n",
			want:     "A=1\nFOO=new\nB=2\n",
		},
		{
			name:     "Idempotent",
			existing: "FOO=new\n",
			body:     "FOO=new\n",
			want:     "FOO=new\n",
		},
		{
			name:     "AddsNewlineBeforeAppend",
			existing: "A=1",
			body:     "FOO=bar\n",
			want:     "A=1\nFOO=bar\n",
		},
		{
			name:     "CreatesFromEmpty",
			existing: "",
			body:     "FOO=bar\n",
			want:     "FOO=bar\n",
		},
		{
			name:     "KeyWithNoSeparatorAppendsDistinct",
			existing: "keepme\n",
			body:     "addme\n",
			want:     "keepme\naddme\n",
		},
		{
			name:     "KeyWithNoSeparatorReplacesExact",
			existing: "solo\n",
			body:     "solo\n",
			want:     "solo\n",
		},
		{
			name:     "NoSeparatorDoesNotMatchKeyed",
			existing: "FOO=1\n",
			body:     "FOO\n",
			want:     "FOO=1\nFOO\n",
		},
		{
			name: "MultiLineRejected",
			body: "A=1\nB=2\n",
			err:  "single line",
		},
		{
			name:     "EmptyRejected",
			existing: "x\n",
			body:     "\n",
			err:      "requires content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ensureLine([]byte(tt.existing), []byte(tt.body))

			if tt.err != "" {
				assert.ErrorContains(t, err, tt.err)
				return
			}

			assert.NilError(t, err)
			assert.Equal(t, string(got), tt.want)
		})
	}

	t.Run("MultipleKeysAccumulate", func(t *testing.T) {
		a, err := ensureLine([]byte("base\n"), []byte("FOO=1\n"))
		assert.NilError(t, err)

		b, err := ensureLine(a, []byte("BAR=2\n"))
		assert.NilError(t, err)
		assert.Equal(t, string(b), "base\nFOO=1\nBAR=2\n")

		c, err := ensureLine(b, []byte("FOO=9\n"))
		assert.NilError(t, err)
		assert.Equal(t, string(c), "base\nFOO=9\nBAR=2\n")
	})
}

func TestApplyLineinfile(t *testing.T) {
	t.Run("CreatesWhenAbsent", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "zshenv")

		writeManifest(t, source, lineinfileManifest(dest, "\"export FOO=1\\n\""))
		apply(t, Options{Source: source})

		assert.Equal(t, readFile(t, dest), "export FOO=1\n")
	})

	t.Run("ReplaceExistingWithoutForce", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "zshenv")
		write(t, dest, "FOO=old\n")

		writeManifest(t, source, lineinfileManifest(dest, "\"FOO=new\\n\""))
		apply(t, Options{Source: source})

		assert.Equal(t, readFile(t, dest), "FOO=new\n")
	})

	t.Run("ZshenvExportIdempotency", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "zshenv")

		writeManifest(t, source, lineinfileManifest(dest, "\"export FOO=1\\n\""))
		apply(t, Options{Source: source})

		writeManifest(t, source, lineinfileManifest(dest, "\"export FOO=2\\n\""))
		apply(t, Options{Source: source})

		got := readFile(t, dest)
		assert.Equal(t, got, "export FOO=2\n")
		assert.Equal(t, strings.Count(got, "export FOO="), 1)
	})

	t.Run("Idempotent", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "zshenv")
		write(t, dest, "A=1\n")

		writeManifest(t, source, lineinfileManifest(dest, "\"FOO=bar\\n\""))
		apply(t, Options{Source: source})
		first := readFile(t, dest)
		apply(t, Options{Source: source})
		second := readFile(t, dest)

		assert.Equal(t, first, second)
	})

	t.Run("PreservesExistingMode", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "zshenv")
		write(t, dest, "FOO=old\n")
		assert.NilError(t, os.Chmod(dest, 0o600))

		writeManifest(t, source, lineinfileManifest(dest, "\"FOO=new\\n\""))
		apply(t, Options{Source: source})

		assert.Equal(t, mode(t, dest), os.FileMode(0o600))
	})

	t.Run("MultiLineContentRejected", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "zshenv")
		write(t, dest, "base\n")

		writeManifest(t, source, lineinfileManifest(dest, "\"A=1\\nB=2\\n\""))
		output := applyErr(t, Options{Source: source})

		assert.Equal(t, readFile(t, dest), "base\n")
		assert.Assert(t, strings.Contains(output, "single line"))
	})
}
