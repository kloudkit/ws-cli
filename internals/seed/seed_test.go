package seed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/secrets"
	"gotest.tools/v3/assert"
)

const testMaster = "ws-seed-test-master-key-0123456789"

func setEnv(t *testing.T, home string) {
	t.Setenv("HOME", home)
	t.Setenv("WS__INTERNAL_ENV_REFERENCE", filepath.Join(t.TempDir(), "absent.yaml"))
}

func write(t *testing.T, path, content string) {
	t.Helper()
	assert.NilError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NilError(t, os.WriteFile(path, []byte(content), 0o644))
}

func writeManifest(t *testing.T, source, body string) {
	t.Helper()
	write(t, filepath.Join(source, ManifestName), "version: v1\n"+body)
}

func rhyming(source, dest string) string {
	return filepath.Join(source, strings.TrimPrefix(dest, "/"))
}

func apply(t *testing.T, opts Options) string {
	t.Helper()
	var buffer bytes.Buffer
	opts.Out = &buffer
	assert.NilError(t, Apply(opts))
	return buffer.String()
}

func applyErr(t *testing.T, opts Options) string {
	t.Helper()
	var buffer bytes.Buffer
	opts.Out = &buffer
	assert.Assert(t, Apply(opts) != nil)
	return buffer.String()
}

func encrypt(t *testing.T, plaintext, key string) string {
	t.Helper()
	master, err := secrets.ResolveMasterKey(key)
	assert.NilError(t, err)
	encrypted, err := secrets.Encrypt([]byte(plaintext), master)
	assert.NilError(t, err)
	return encrypted
}

func mode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	assert.NilError(t, err)
	return info.Mode().Perm()
}

func TestApplyMirror(t *testing.T) {
	t.Run("Verbatim", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "file.txt")

		write(t, rhyming(source, dest), "verbatim\n")

		apply(t, Options{Source: source})

		got, err := os.ReadFile(dest)
		assert.NilError(t, err)
		assert.Equal(t, string(got), "verbatim\n")
	})

	t.Run("ManifestNotProjected", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		writeManifest(t, source, "")

		plan, err := BuildPlan(source, false)
		assert.NilError(t, err)
		assert.Equal(t, len(plan.Ops), 0)
	})
}

func TestApplyInline(t *testing.T) {
	t.Run("Literal", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "out.txt")

		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    mode: \"0o640\"\n    content: \"inline\\n\"\n", dest))

		apply(t, Options{Source: source})

		got, err := os.ReadFile(dest)
		assert.NilError(t, err)
		assert.Equal(t, string(got), "inline\n")
		assert.Equal(t, mode(t, dest), os.FileMode(0o640))
	})

	t.Run("CopyOnlyMissingSourceWarns", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "ghost.txt")

		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    mode: \"0o644\"\n", dest))

		output := applyErr(t, Options{Source: source})

		assert.Assert(t, !fileExists(dest))
		assert.Assert(t, strings.Contains(output, "Skipping ["+dest+"] (no source available)"))
	})

	t.Run("SourceUnreadableReported", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "out.txt")

		assert.NilError(t, os.MkdirAll(rhyming(source, dest), 0o755))
		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    mode: \"0o644\"\n", dest))

		output := applyErr(t, Options{Source: source})

		assert.Assert(t, strings.Contains(output, "source unreadable"))
	})
}

func TestApplyOps(t *testing.T) {
	t.Run("Append", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "list.txt")
		write(t, dest, "base\n")

		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    op: append\n    force: true\n    content: \"added\\n\"\n", dest))

		apply(t, Options{Source: source})

		got, err := os.ReadFile(dest)
		assert.NilError(t, err)
		assert.Equal(t, string(got), "base\nadded\n")
	})

	t.Run("Prepend", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "list.txt")
		write(t, dest, "base\n")

		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    op: prepend\n    force: true\n    content: \"added\\n\"\n", dest))

		apply(t, Options{Source: source})

		got, err := os.ReadFile(dest)
		assert.NilError(t, err)
		assert.Equal(t, string(got), "added\nbase\n")
	})

	t.Run("MergeJSON", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "config.json")
		write(t, dest, `{"a":1,"list":[1,2,3]}`)

		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    op: merge\n    force: true\n    content: '{\"list\":[9],\"b\":2}'\n", dest))

		apply(t, Options{Source: source})

		out := decodeBack(t, []byte(readFile(t, dest)), dest)
		assert.Equal(t, out["a"], json.Number("1"))
		assert.Equal(t, out["b"], json.Number("2"))
		list, ok := out["list"].([]any)
		assert.Assert(t, ok)
		assert.Equal(t, len(list), 1)
	})

	t.Run("MergePreservesExistingMode", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "config.json")
		write(t, dest, `{"a":1}`)
		assert.NilError(t, os.Chmod(dest, 0o600))

		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    op: merge\n    force: true\n    content: '{\"b\":2}'\n", dest))

		apply(t, Options{Source: source})

		assert.Equal(t, mode(t, dest), os.FileMode(0o600))
	})

	t.Run("MergeScalarVsMapConflictLeavesDestUnchanged", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "config.json")
		write(t, dest, `{"k":"scalar"}`)

		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    op: merge\n    force: true\n    content: '{\"k\":{\"nested\":1}}'\n", dest))

		output := applyErr(t, Options{Source: source})

		assert.Equal(t, readFile(t, dest), `{"k":"scalar"}`)
		assert.Assert(t, strings.Contains(output, "merge conflict at key"))
	})
}

func TestApplyTemplate(t *testing.T) {
	t.Run("WsTokensAndSecret", func(t *testing.T) {
		home := t.TempDir()
		setEnv(t, home)
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "rendered.txt")

		ciphertext := encrypt(t, "S3CR3T", testMaster)
		writeManifest(t, source, fmt.Sprintf(
			"secrets:\n  TOK: %s\nseeds:\n  %s:\n    template: true\n    content: \"${ws_home}|${ws_user}|${secrets.TOK}\\n\"\n",
			ciphertext, dest,
		))

		apply(t, Options{Source: source, MasterKey: testMaster})

		current, err := user.Current()
		assert.NilError(t, err)
		assert.Equal(t, readFile(t, dest), fmt.Sprintf("%s|%s|S3CR3T\n", home, current.Username))
	})

	t.Run("UnknownTokenFailsLoud", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "rendered.txt")

		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    template: true\n    content: \"${bogus}\\n\"\n", dest))

		output := applyErr(t, Options{Source: source})

		assert.Assert(t, !fileExists(dest))
		assert.Assert(t, strings.Contains(output, "unknown template token ${bogus}"))
	})

	t.Run("SecretBearingFloor", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "secret.txt")

		ciphertext := encrypt(t, "TOP-SECRET-VALUE", testMaster)
		writeManifest(t, source, fmt.Sprintf(
			"secrets:\n  TOK: %s\nseeds:\n  %s:\n    template: true\n    content: \"${secrets.TOK}\\n\"\n",
			ciphertext, dest,
		))

		output := apply(t, Options{Source: source, MasterKey: testMaster})

		assert.Equal(t, readFile(t, dest), "TOP-SECRET-VALUE\n")
		assert.Equal(t, mode(t, dest), os.FileMode(0o600))
		assert.Assert(t, !strings.Contains(output, "TOP-SECRET-VALUE"))
	})

	t.Run("SecretBearingFailureOutputScrubbed", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "conflict.json")
		write(t, dest, `{"key":"scalar"}`)

		ciphertext := encrypt(t, "TOP-SECRET-VALUE", testMaster)
		writeManifest(t, source, fmt.Sprintf(
			"secrets:\n  TOK: %s\nseeds:\n  %s:\n    template: true\n    op: merge\n    force: true\n    content: '{\"key\":{\"n\":\"${secrets.TOK}\"}}'\n",
			ciphertext, dest,
		))

		output := applyErr(t, Options{Source: source, MasterKey: testMaster})

		assert.Equal(t, readFile(t, dest), `{"key":"scalar"}`)
		assert.Assert(t, strings.Contains(output, "merge conflict at key"))
		assert.Assert(t, !strings.Contains(output, "TOP-SECRET-VALUE"))
	})
}

func TestApplySecrets(t *testing.T) {
	t.Run("WholeFile", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "id_key")

		write(t, rhyming(source, dest), encrypt(t, "PRIVATE-KEY-BODY\n", testMaster))
		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    secret: true\n", dest))

		apply(t, Options{Source: source, MasterKey: testMaster})

		assert.Equal(t, readFile(t, dest), "PRIVATE-KEY-BODY\n")
		assert.Equal(t, mode(t, dest), os.FileMode(0o600))
	})

	t.Run("FailClosedOnBadDecrypt", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "id_key")

		write(t, rhyming(source, dest), encrypt(t, "PRIVATE", "a-totally-different-master-key-99"))
		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    secret: true\n", dest))

		output := applyErr(t, Options{Source: source, MasterKey: testMaster})

		assert.Assert(t, !fileExists(dest))
		assert.Assert(t, strings.Contains(output, "Skipping ["+dest+"] (decrypt failed)"))
		assert.Assert(t, !strings.Contains(output, "PRIVATE"))
	})

	t.Run("SecretFreeManifestNeedsNoKey", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "plain.txt")

		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    mode: \"0o644\"\n    content: \"plain\\n\"\n", dest))

		apply(t, Options{Source: source})

		assert.Equal(t, readFile(t, dest), "plain\n")
	})

	t.Run("MissingKeyFailsClosedButNonSecretApplies", func(t *testing.T) {
		setEnv(t, t.TempDir())
		t.Setenv("WS_SECRETS_MASTER_KEY", "")
		source := t.TempDir()
		target := t.TempDir()
		secretDest := filepath.Join(target, "secret.txt")
		plainDest := filepath.Join(target, "plain.txt")

		write(t, rhyming(source, secretDest), encrypt(t, "X", testMaster))
		writeManifest(t, source, fmt.Sprintf(
			"seeds:\n  %s:\n    secret: true\n  %s:\n    mode: \"0o644\"\n    content: \"plain\\n\"\n",
			secretDest, plainDest,
		))

		output := applyErr(t, Options{Source: source})

		assert.Assert(t, !fileExists(secretDest))
		assert.Equal(t, readFile(t, plainDest), "plain\n")
		assert.Assert(t, strings.Contains(output, "master key unavailable"))
	})
}

func TestApplyOwnership(t *testing.T) {
	t.Run("OwnedAncestorAllows", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "nested", "deep", "out.txt")

		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    mode: \"0o644\"\n    content: \"ok\\n\"\n", dest))

		apply(t, Options{Source: source})

		assert.Equal(t, readFile(t, dest), "ok\n")
	})

	t.Run("NonOwnedAncestorSkips", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("running as root: every ancestor is writable")
		}

		setEnv(t, t.TempDir())
		source := t.TempDir()
		dest := "/etc/ws-seed-test-should-not-write/out.txt"

		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    mode: \"0o644\"\n    content: \"x\\n\"\n", dest))

		output := applyErr(t, Options{Source: source})

		assert.Assert(t, !fileExists(dest))
		assert.Assert(t, strings.Contains(output, "Skipping ["+dest+"] (destination not owned)"))
	})
}

func TestApplyPrecedence(t *testing.T) {
	t.Run("ManifestWinsOverBare", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "shared.txt")

		write(t, rhyming(source, dest), "BARE\n")
		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    mode: \"0o600\"\n    content: \"MANIFEST\\n\"\n", dest))

		apply(t, Options{Source: source})

		assert.Equal(t, readFile(t, dest), "MANIFEST\n")
		assert.Equal(t, mode(t, dest), os.FileMode(0o600))
	})

	t.Run("SecretSuppressesRawProjection", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "key")

		ciphertext := encrypt(t, "PLAINTEXT\n", testMaster)
		write(t, rhyming(source, dest), ciphertext)
		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    secret: true\n", dest))

		apply(t, Options{Source: source, MasterKey: testMaster})

		got := readFile(t, dest)
		assert.Equal(t, got, "PLAINTEXT\n")
		assert.Assert(t, got != ciphertext)
		assert.Equal(t, mode(t, dest), os.FileMode(0o600))
	})
}

func TestApplyForceMatrix(t *testing.T) {
	manifest := func(dest, op, content string) string {
		return fmt.Sprintf("seeds:\n  %s:\n    op: %s\n    mode: \"0o644\"\n    content: %s\n", dest, op, content)
	}

	t.Run("AbsentWrites", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "f.txt")

		writeManifest(t, source, manifest(dest, "copy", "\"v1\\n\""))
		apply(t, Options{Source: source})

		assert.Equal(t, readFile(t, dest), "v1\n")
	})

	t.Run("ExistsForceFalseSkips", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "f.txt")
		write(t, dest, "orig\n")

		writeManifest(t, source, manifest(dest, "copy", "\"v2\\n\""))
		apply(t, Options{Source: source})

		assert.Equal(t, readFile(t, dest), "orig\n")
	})

	t.Run("ExistsForceTrueOverwrites", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "f.txt")
		write(t, dest, "orig\n")

		writeManifest(t, source, manifest(dest, "copy", "\"v2\\n\""))
		apply(t, Options{Source: source, Force: true})

		assert.Equal(t, readFile(t, dest), "v2\n")
	})

	t.Run("MergeForceFalseExistsSkips", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "f.json")
		write(t, dest, `{"a":1}`)

		writeManifest(t, source, manifest(dest, "merge", "'{\"b\":2}'"))
		apply(t, Options{Source: source})

		assert.Equal(t, readFile(t, dest), `{"a":1}`)
	})
}

func TestWriteAtomicSymlink(t *testing.T) {
	t.Run("EscapingComponentRefused", func(t *testing.T) {
		anchor := t.TempDir()
		outside := t.TempDir()
		assert.NilError(t, os.Symlink(outside, filepath.Join(anchor, "evil")))

		dest := filepath.Join(anchor, "evil", "file.txt")
		err := writeAtomic(anchor, dest, []byte("x"), 0o644)

		assert.Assert(t, err != nil)
		assert.Assert(t, !fileExists(filepath.Join(outside, "file.txt")))
	})

	t.Run("FinalComponentSymlinkRefused", func(t *testing.T) {
		anchor := t.TempDir()
		outside := filepath.Join(t.TempDir(), "target.txt")
		assert.NilError(t, os.Symlink(outside, filepath.Join(anchor, "link")))

		dest := filepath.Join(anchor, "link")
		err := writeAtomic(anchor, dest, []byte("x"), 0o644)

		assert.ErrorContains(t, err, "refusing to write through symlink")
		assert.Assert(t, !fileExists(outside))
	})
}

func fileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.NilError(t, err)
	return string(data)
}
