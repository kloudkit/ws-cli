package seed

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func blockManifest(dest, content string) string {
	return fmt.Sprintf("seeds:\n  %s:\n    op: block\n    content: %s\n", dest, content)
}

func TestEnsureBlock(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		body     string
		comment  string
		want     string
		err      string
	}{
		{
			name: "AppendsWhenAbsent",
			body: "export FOO=1\n",
			want: "# >>> ws-seed >>>\nexport FOO=1\n# <<< ws-seed <<<\n",
		},
		{
			name:    "CustomCommentPrefix",
			body:    "const x = 1;\n",
			comment: "//",
			want:    "// >>> ws-seed >>>\nconst x = 1;\n// <<< ws-seed <<<\n",
		},
		{
			name:     "CustomCommentReplacesBody",
			existing: "// >>> ws-seed >>>\nold\n// <<< ws-seed <<<\n",
			body:     "new\n",
			comment:  "//",
			want:     "// >>> ws-seed >>>\nnew\n// <<< ws-seed <<<\n",
		},
		{
			name:     "AppendsAfterExisting",
			existing: "base\n",
			body:     "line\n",
			want:     "base\n# >>> ws-seed >>>\nline\n# <<< ws-seed <<<\n",
		},
		{
			name:     "AddsNewlineBeforeMarker",
			existing: "base",
			body:     "line\n",
			want:     "base\n# >>> ws-seed >>>\nline\n# <<< ws-seed <<<\n",
		},
		{
			name:     "ReplacesBodyPreservingSurround",
			existing: "head\n# >>> ws-seed >>>\nold\n# <<< ws-seed <<<\ntail\n",
			body:     "new\n",
			want:     "head\n# >>> ws-seed >>>\nnew\n# <<< ws-seed <<<\ntail\n",
		},
		{
			name:     "DuplicateBeginRejected",
			existing: "# >>> ws-seed >>>\n# >>> ws-seed >>>\n# <<< ws-seed <<<\n",
			body:     "x\n",
			err:      "duplicate begin marker",
		},
		{
			name:     "EndBeforeBeginRejected",
			existing: "# <<< ws-seed <<<\nbody\n# >>> ws-seed >>>\n",
			body:     "x\n",
			err:      "markers out of order",
		},
		{
			name:     "BeginWithoutEndRejected",
			existing: "# >>> ws-seed >>>\norphan\n",
			body:     "x\n",
			err:      "markers out of order",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ensureBlock([]byte(tt.existing), []byte(tt.body), tt.comment)

			if tt.err != "" {
				assert.ErrorContains(t, err, tt.err)
				return
			}

			assert.NilError(t, err)
			assert.Equal(t, string(got), tt.want)
		})
	}
}

func TestApplyBlock(t *testing.T) {
	t.Run("CreatesWhenAbsent", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "zshenv")

		writeManifest(t, source, blockManifest(dest, "\"export FOO=1\\n\""))
		apply(t, Options{Source: source})

		assert.Equal(t, readFile(t, dest), "# >>> ws-seed >>>\nexport FOO=1\n# <<< ws-seed <<<\n")
	})

	t.Run("AppendsToExistingWithoutForce", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "zshenv")
		write(t, dest, "base\n")

		writeManifest(t, source, blockManifest(dest, "\"line\\n\""))
		apply(t, Options{Source: source})

		assert.Equal(t, readFile(t, dest), "base\n# >>> ws-seed >>>\nline\n# <<< ws-seed <<<\n")
	})

	t.Run("Idempotent", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "zshenv")
		write(t, dest, "base\n")

		writeManifest(t, source, blockManifest(dest, "\"line\\n\""))
		apply(t, Options{Source: source})
		first := readFile(t, dest)
		apply(t, Options{Source: source})
		second := readFile(t, dest)

		assert.Equal(t, first, second)
		assert.Equal(t, strings.Count(second, blockBegin), 1)
	})

	t.Run("ReplacesBodyOnChange", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "zshenv")
		write(t, dest, "head\n")

		writeManifest(t, source, blockManifest(dest, "\"v1\\n\""))
		apply(t, Options{Source: source})

		writeManifest(t, source, blockManifest(dest, "\"v2\\n\""))
		apply(t, Options{Source: source})

		got := readFile(t, dest)
		assert.Equal(t, got, "head\n# >>> ws-seed >>>\nv2\n# <<< ws-seed <<<\n")
		assert.Equal(t, strings.Count(got, blockBegin), 1)
	})

	t.Run("PreservesExistingMode", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "zshenv")
		write(t, dest, "base\n")
		assert.NilError(t, os.Chmod(dest, 0o600))

		writeManifest(t, source, blockManifest(dest, "\"line\\n\""))
		apply(t, Options{Source: source})

		assert.Equal(t, mode(t, dest), os.FileMode(0o600))
	})

	t.Run("CustomCommentMarker", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "app.js")

		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    op: block\n    comment: \"//\"\n    content: \"const x = 1;\\n\"\n", dest))
		apply(t, Options{Source: source})

		assert.Equal(t, readFile(t, dest), "// >>> ws-seed >>>\nconst x = 1;\n// <<< ws-seed <<<\n")
	})

	t.Run("MalformedMarkersFailClosed", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "zshenv")
		write(t, dest, "# >>> ws-seed >>>\norphan\n")

		writeManifest(t, source, blockManifest(dest, "\"line\\n\""))
		output := applyErr(t, Options{Source: source})

		assert.Equal(t, readFile(t, dest), "# >>> ws-seed >>>\norphan\n")
		assert.Assert(t, strings.Contains(output, "malformed managed block"))
	})
}
