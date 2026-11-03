package seed

import (
	"bytes"
	"cmp"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/secrets"
	"gotest.tools/v3/assert"
)

const (
	testMaster  = "ws-seed-test-master-key-0123456789"
	otherMaster = "a-totally-different-master-key-99"
)

func setEnv(t *testing.T, home string) {
	t.Setenv("HOME", home)
	t.Setenv("WS_SECRETS_MASTER_KEY", "")
	t.Setenv("WS__INTERNAL_ENV_REFERENCE", filepath.Join(t.TempDir(), "absent.yaml"))
}

func newSource(t *testing.T) string {
	setEnv(t, t.TempDir())
	return t.TempDir()
}

func write(t *testing.T, path, content string) {
	t.Helper()
	assert.NilError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NilError(t, os.WriteFile(path, []byte(content), 0o644))
}

func writeManifest(t *testing.T, source, body string) {
	t.Helper()
	write(t, manifestPath(source), "version: v1\n"+body)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.NilError(t, err)
	return string(data)
}

func fileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func mode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	assert.NilError(t, err)
	return info.Mode().Perm()
}

func encrypt(t *testing.T, plaintext, key string) string {
	t.Helper()
	master, err := secrets.ResolveMasterKey(key)
	assert.NilError(t, err)
	encrypted, err := secrets.Encrypt([]byte(plaintext), master)
	assert.NilError(t, err)
	return encrypted
}

func runApply(t *testing.T, opts Options, wantErr bool) string {
	t.Helper()
	var buffer bytes.Buffer
	opts.Out = &buffer
	assert.Equal(t, Apply(opts) != nil, wantErr, buffer.String())
	return buffer.String()
}

func manifestFor(op, extra, content string) string {
	return fmt.Sprintf("seeds:\n  DEST:\n    op: %s\n%s    content: %s\n", op, extra, content)
}

func TestApply(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		existing string
		perm     os.FileMode
		mirror   string
		manifest string
		rerun    string
		secret   string
		secretBy string
		key      bool
		force    bool
		fails    bool
		want     string
		mode     os.FileMode
		output   string
		hidden   string
	}{
		{name: "MirrorVerbatim", mirror: "verbatim\n", want: "verbatim\n"},
		{name: "InlineLiteral", manifest: "seeds:\n  DEST:\n    mode: \"0o640\"\n    content: \"inline\\n\"\n", want: "inline\n", mode: 0o640},
		{name: "CopyOnlyMissingSourceWarns", manifest: "seeds:\n  DEST:\n    mode: \"0o644\"\n", fails: true, output: "Skipping [DEST] (no source available)"},
		{name: "Append", existing: "base\n", manifest: manifestFor("append", "    force: true\n", `"added\n"`), want: "base\nadded\n"},
		{name: "Prepend", existing: "base\n", manifest: manifestFor("prepend", "    force: true\n", `"added\n"`), want: "added\nbase\n"},
		{
			name: "MergeJSON", file: "config.json", existing: `{"a":1,"list":[1,2,3]}`,
			manifest: manifestFor("merge", "    force: true\n", `'{"list":[9],"b":2}'`),
			want:     "{\n  \"a\": 1,\n  \"b\": 2,\n  \"list\": [\n    9\n  ]\n}\n",
		},
		{name: "MergePreservesExistingMode", file: "config.json", existing: `{"a":1}`, perm: 0o600, manifest: manifestFor("merge", "    force: true\n", `'{"b":2}'`), mode: 0o600},
		{
			name: "MergeConflictLeavesDestUnchanged", file: "config.json", existing: `{"k":"scalar"}`,
			manifest: manifestFor("merge", "    force: true\n", `'{"k":{"nested":1}}'`),
			fails:    true, want: `{"k":"scalar"}`, output: "merge conflict at key",
		},
		{name: "MergeNullExisting", file: "config.json", existing: "null", manifest: manifestFor("merge", "    force: true\n", `'{"b":2}'`), want: "{\n  \"b\": 2\n}\n"},
		{name: "TemplateUnknownTokenFailsLoud", manifest: "seeds:\n  DEST:\n    template: true\n    content: \"${bogus}\\n\"\n", fails: true, output: "unknown template token ${bogus}"},
		{
			name: "TemplateSecretBearingFloor", secret: "TOP-SECRET-VALUE", key: true,
			manifest: "secrets:\n  TOK: CIPHER\nseeds:\n  DEST:\n    template: true\n    content: \"${secrets.TOK}\\n\"\n",
			want:     "TOP-SECRET-VALUE\n", mode: 0o600, hidden: "TOP-SECRET-VALUE",
		},
		{
			name: "TemplateSecretFailureScrubbed", file: "conflict.json", existing: `{"key":"scalar"}`, secret: "TOP-SECRET-VALUE", key: true,
			manifest: "secrets:\n  TOK: CIPHER\n" + manifestFor("merge", "    template: true\n    force: true\n", `'{"key":{"n":"${secrets.TOK}"}}'`),
			fails:    true, want: `{"key":"scalar"}`, output: "merge conflict at key", hidden: "TOP-SECRET-VALUE",
		},
		{name: "SecretWholeFile", secret: "PRIVATE-KEY-BODY\n", key: true, mirror: "CIPHER", manifest: "seeds:\n  DEST:\n    secret: true\n", want: "PRIVATE-KEY-BODY\n", mode: 0o600},
		{
			name: "SecretBadDecryptFailsClosed", secret: "PRIVATE", secretBy: otherMaster, key: true, mirror: "CIPHER",
			manifest: "seeds:\n  DEST:\n    secret: true\n", fails: true, output: "Skipping [DEST] (decrypt failed)", hidden: "PRIVATE",
		},
		{name: "SecretMissingKeyFailsClosed", secret: "X", mirror: "CIPHER", manifest: "seeds:\n  DEST:\n    secret: true\n", fails: true, output: "master key unavailable"},
		{name: "SecretFreeManifestNeedsNoKey", manifest: "seeds:\n  DEST:\n    content: \"plain\\n\"\n", want: "plain\n"},
		{name: "OwnedAncestorAllows", file: "nested/deep/out.txt", manifest: "seeds:\n  DEST:\n    content: \"ok\\n\"\n", want: "ok\n"},
		{name: "ManifestWinsOverBare", mirror: "BARE\n", manifest: "seeds:\n  DEST:\n    mode: \"0o600\"\n    content: \"MANIFEST\\n\"\n", want: "MANIFEST\n", mode: 0o600},
		{name: "ForceAbsentWrites", manifest: manifestFor("copy", "    mode: \"0o644\"\n", `"v1\n"`), want: "v1\n"},
		{name: "ForceFalseSkipsExisting", existing: "orig\n", manifest: manifestFor("copy", "    mode: \"0o644\"\n", `"v2\n"`), want: "orig\n", output: "Skipping [DEST] (exists)"},
		{name: "ForceTrueOverwrites", existing: "orig\n", force: true, manifest: manifestFor("copy", "    mode: \"0o644\"\n", `"v2\n"`), want: "v2\n"},
		{name: "MergeForceFalseSkips", file: "f.json", existing: `{"a":1}`, manifest: manifestFor("merge", "", `'{"b":2}'`), want: `{"a":1}`},
		{name: "BlockCreates", manifest: manifestFor("block", "", `"export FOO=1\n"`), want: "# >>> ws-seed >>>\nexport FOO=1\n# <<< ws-seed <<<\n"},
		{name: "BlockAppendsWithoutForce", existing: "base\n", manifest: manifestFor("block", "", `"line\n"`), want: "base\n# >>> ws-seed >>>\nline\n# <<< ws-seed <<<\n"},
		{
			name: "BlockIdempotent", existing: "base\n", manifest: manifestFor("block", "", `"line\n"`), rerun: manifestFor("block", "", `"line\n"`),
			want: "base\n# >>> ws-seed >>>\nline\n# <<< ws-seed <<<\n",
		},
		{
			name: "BlockReplacesBodyOnChange", existing: "head\n", manifest: manifestFor("block", "", `"v1\n"`), rerun: manifestFor("block", "", `"v2\n"`),
			want: "head\n# >>> ws-seed >>>\nv2\n# <<< ws-seed <<<\n",
		},
		{name: "BlockPreservesMode", existing: "base\n", perm: 0o600, manifest: manifestFor("block", "", `"line\n"`), mode: 0o600},
		{name: "BlockCustomComment", manifest: manifestFor("block", "    comment: \"//\"\n", `"const x = 1;\n"`), want: "// >>> ws-seed >>>\nconst x = 1;\n// <<< ws-seed <<<\n"},
		{
			name: "BlockMalformedFailsClosed", existing: "# >>> ws-seed >>>\norphan\n", manifest: manifestFor("block", "", `"line\n"`),
			fails: true, want: "# >>> ws-seed >>>\norphan\n", output: "malformed managed block",
		},
		{name: "LineCreates", manifest: manifestFor("lineinfile", "", `"export FOO=1\n"`), want: "export FOO=1\n"},
		{name: "LineReplacesWithoutForce", existing: "FOO=old\n", manifest: manifestFor("lineinfile", "", `"FOO=new\n"`), want: "FOO=new\n"},
		{
			name: "LineExportRewrites", manifest: manifestFor("lineinfile", "", `"export FOO=1\n"`), rerun: manifestFor("lineinfile", "", `"export FOO=2\n"`),
			want: "export FOO=2\n",
		},
		{name: "LineIdempotent", existing: "A=1\n", manifest: manifestFor("lineinfile", "", `"FOO=bar\n"`), rerun: manifestFor("lineinfile", "", `"FOO=bar\n"`), want: "A=1\nFOO=bar\n"},
		{name: "LinePreservesMode", existing: "FOO=old\n", perm: 0o600, manifest: manifestFor("lineinfile", "", `"FOO=new\n"`), mode: 0o600},
		{
			name: "LineMultiLineRejected", existing: "base\n", manifest: manifestFor("lineinfile", "", `"A=1\nB=2\n"`),
			fails: true, want: "base\n", output: "single line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := newSource(t)
			dest := filepath.Join(t.TempDir(), cmp.Or(tt.file, "out.txt"))

			cipher := ""
			if tt.secret != "" {
				cipher = encrypt(t, tt.secret, cmp.Or(tt.secretBy, testMaster))
			}
			fill := strings.NewReplacer("DEST", dest, "CIPHER", cipher).Replace

			if tt.existing != "" {
				write(t, dest, tt.existing)
			}
			if tt.perm != 0 {
				assert.NilError(t, os.Chmod(dest, tt.perm))
			}
			if tt.mirror != "" {
				write(t, rhymingSource(source, dest), fill(tt.mirror))
			}
			if tt.manifest != "" {
				writeManifest(t, source, fill(tt.manifest))
			}

			opts := Options{Source: source, Force: tt.force}
			if tt.key {
				opts.MasterKey = testMaster
			}

			output := runApply(t, opts, tt.fails)
			if tt.rerun != "" {
				writeManifest(t, source, fill(tt.rerun))
				output = runApply(t, opts, tt.fails)
			}

			switch {
			case tt.want != "":
				assert.Equal(t, readFile(t, dest), tt.want)
			case tt.mode == 0:
				assert.Assert(t, !fileExists(dest))
			}

			if tt.mode != 0 {
				assert.Equal(t, mode(t, dest), tt.mode)
			}
			if tt.output != "" {
				assert.Assert(t, strings.Contains(output, fill(tt.output)), output)
			}
			if tt.hidden != "" {
				assert.Assert(t, !strings.Contains(output, tt.hidden))
			}
		})
	}
}

func TestApplySpecialCases(t *testing.T) {
	t.Run("TemplateWsTokensAndSecret", func(t *testing.T) {
		home := t.TempDir()
		setEnv(t, home)
		source := t.TempDir()
		dest := filepath.Join(t.TempDir(), "rendered.txt")
		writeManifest(t, source, fmt.Sprintf(
			"secrets:\n  TOK: %s\nseeds:\n  %s:\n    template: true\n    content: \"${ws_home}|${ws_user}|${secrets.TOK}\\n\"\n",
			encrypt(t, "S3CR3T", testMaster), dest,
		))

		runApply(t, Options{Source: source, MasterKey: testMaster}, false)

		current, err := user.Current()
		assert.NilError(t, err)
		assert.Equal(t, readFile(t, dest), fmt.Sprintf("%s|%s|S3CR3T\n", home, current.Username))
	})

	t.Run("SourceUnreadableReported", func(t *testing.T) {
		source := newSource(t)
		dest := filepath.Join(t.TempDir(), "out.txt")
		assert.NilError(t, os.MkdirAll(rhymingSource(source, dest), 0o755))
		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    mode: \"0o644\"\n", dest))

		assert.Assert(t, strings.Contains(runApply(t, Options{Source: source}, true), "source unreadable"))
	})

	t.Run("MissingKeyStillAppliesNonSecret", func(t *testing.T) {
		source := newSource(t)
		target := t.TempDir()
		secretDest, plainDest := filepath.Join(target, "secret.txt"), filepath.Join(target, "plain.txt")
		write(t, rhymingSource(source, secretDest), encrypt(t, "X", testMaster))
		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    secret: true\n  %s:\n    content: \"plain\\n\"\n", secretDest, plainDest))

		output := runApply(t, Options{Source: source}, true)

		assert.Assert(t, !fileExists(secretDest))
		assert.Equal(t, readFile(t, plainDest), "plain\n")
		assert.Assert(t, strings.Contains(output, "master key unavailable"))
	})

	t.Run("NonOwnedAncestorSkips", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("running as root: every ancestor is writable")
		}

		source := newSource(t)
		dest := "/etc/ws-seed-test-should-not-write/out.txt"
		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    content: \"x\\n\"\n", dest))

		output := runApply(t, Options{Source: source}, true)

		assert.Assert(t, !fileExists(dest))
		assert.Assert(t, strings.Contains(output, "Skipping ["+dest+"] (destination not owned)"))
	})

	t.Run("ManifestNotProjected", func(t *testing.T) {
		source := newSource(t)
		writeManifest(t, source, "")

		plan, err := BuildPlan(source, false)
		assert.NilError(t, err)
		assert.Equal(t, len(plan.Ops), 0)
	})
}

func TestWriteAtomicSymlink(t *testing.T) {
	t.Run("EscapingComponentRefused", func(t *testing.T) {
		anchor, outside := t.TempDir(), t.TempDir()
		assert.NilError(t, os.Symlink(outside, filepath.Join(anchor, "evil")))

		assert.Assert(t, writeAtomic(anchor, filepath.Join(anchor, "evil", "file.txt"), []byte("x"), 0o644) != nil)
		assert.Assert(t, !fileExists(filepath.Join(outside, "file.txt")))
	})

	t.Run("FinalComponentSymlinkRefused", func(t *testing.T) {
		anchor, outside := t.TempDir(), filepath.Join(t.TempDir(), "target.txt")
		assert.NilError(t, os.Symlink(outside, filepath.Join(anchor, "link")))

		err := writeAtomic(anchor, filepath.Join(anchor, "link"), []byte("x"), 0o644)
		assert.ErrorContains(t, err, "refusing to write through symlink")
		assert.Assert(t, !fileExists(outside))
	})
}
