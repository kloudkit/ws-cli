package seed

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/secrets"
	"gotest.tools/v3/assert"
)

const testNewMaster = "ws-seed-rotate-new-master-key-987654"

func runRotate(t *testing.T, opts RotateOptions, wantErr bool) {
	t.Helper()
	var buffer bytes.Buffer
	opts.Out = &buffer
	assert.Equal(t, Rotate(opts) != nil, wantErr, buffer.String())
}

func rotate(t *testing.T, source string, wantErr bool) {
	t.Helper()
	runRotate(t, RotateOptions{Source: source, MasterKey: testMaster, NewMasterKey: testNewMaster}, wantErr)
}

func decryptsUnder(t *testing.T, ciphertext, key, want string) {
	t.Helper()
	master, err := secrets.ResolveMasterKey(key)
	assert.NilError(t, err)
	plain, err := secrets.Decrypt(secrets.NormalizeEncrypted(ciphertext), master)
	if want == "" {
		assert.Assert(t, err != nil)
		return
	}
	assert.NilError(t, err)
	assert.Equal(t, string(plain), want)
}

func rekeyed(t *testing.T, ciphertext, want string) {
	t.Helper()
	decryptsUnder(t, ciphertext, testNewMaster, want)
	decryptsUnder(t, ciphertext, testMaster, "")
}

func loadManifest(t *testing.T, source string) *Manifest {
	t.Helper()
	manifest, err := parseManifest([]byte(readFile(t, manifestPath(source))))
	assert.NilError(t, err)
	return manifest
}

func TestRotate(t *testing.T) {
	t.Run("NewMasterRequired", func(t *testing.T) {
		source := newSource(t)
		writeManifest(t, source, fmt.Sprintf("secrets:\n  TOK: %s\n", encrypt(t, "V", testMaster)))

		runRotate(t, RotateOptions{Source: source, MasterKey: testMaster}, true)
	})

	t.Run("RotatesInlineSecretsMap", func(t *testing.T) {
		source := newSource(t)
		writeManifest(t, source, fmt.Sprintf("secrets:\n  TOK: %s\n", encrypt(t, "TOKENVAL", testMaster)))

		rotate(t, source, false)

		manifest := loadManifest(t, source)
		rekeyed(t, manifest.Secrets["TOK"], "TOKENVAL")
	})

	t.Run("RotatesFileRef", func(t *testing.T) {
		source := newSource(t)
		keyFile := filepath.Join(t.TempDir(), "tok.enc")
		write(t, keyFile, encrypt(t, "FILEVAL", testMaster))
		writeManifest(t, source, fmt.Sprintf("secrets:\n  TOK: file:%s\n", keyFile))

		rotate(t, source, false)

		manifest := loadManifest(t, source)
		assert.Equal(t, manifest.Secrets["TOK"], "file:"+keyFile)
		rekeyed(t, readFile(t, keyFile), "FILEVAL")
	})

	t.Run("RotatesSecretTrueMirror", func(t *testing.T) {
		source := newSource(t)
		dest := filepath.Join(t.TempDir(), "id_key")
		write(t, rhymingSource(source, dest), encrypt(t, "PRIVKEY\n", testMaster))
		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    secret: true\n", dest))

		rotate(t, source, false)

		mirror := readFile(t, rhymingSource(source, dest))
		rekeyed(t, mirror, "PRIVKEY\n")
	})

	t.Run("RotatesSecretTrueInlineContent", func(t *testing.T) {
		source := newSource(t)
		dest := filepath.Join(t.TempDir(), "id_key")
		writeManifest(t, source, fmt.Sprintf(
			"seeds:\n  %s:\n    secret: true\n    content: \"%s\"\n",
			dest, encrypt(t, "INLINEKEY\n", testMaster),
		))

		rotate(t, source, false)

		manifest := loadManifest(t, source)
		rekeyed(t, *manifest.Seeds[dest].Content, "INLINEKEY\n")
	})

	t.Run("PreservesCommentsAndKeyOrder", func(t *testing.T) {
		source := newSource(t)
		manifest := fmt.Sprintf(
			"# managed by ws\nversion: v1\nsecrets:\n  # my token\n  TOK: %s\nseeds:\n  /tmp/keep:\n    content: \"plain\\n\"\n",
			encrypt(t, "V", testMaster),
		)
		write(t, manifestPath(source), manifest)

		rotate(t, source, false)

		after := readFile(t, manifestPath(source))
		assert.Assert(t, strings.Contains(after, "# managed by ws"))
		assert.Assert(t, strings.Contains(after, "# my token"))
		assert.Assert(t, strings.Contains(after, "plain"))
		assert.Assert(t, strings.Index(after, "secrets:") < strings.Index(after, "seeds:"))

		parsed, err := parseManifest([]byte(after))
		assert.NilError(t, err)
		decryptsUnder(t, parsed.Secrets["TOK"], testNewMaster, "V")
	})

	t.Run("WrongOldKeyFailsClosed", func(t *testing.T) {
		source := newSource(t)
		writeManifest(t, source, fmt.Sprintf("secrets:\n  TOK: %s\n", encrypt(t, "V", testMaster)))
		before := readFile(t, manifestPath(source))

		runRotate(t, RotateOptions{Source: source, MasterKey: otherMaster, NewMasterKey: testNewMaster}, true)

		assert.Equal(t, readFile(t, manifestPath(source)), before)
	})

	t.Run("AbsentOldKeyFailsClosed", func(t *testing.T) {
		source := newSource(t)
		writeManifest(t, source, fmt.Sprintf("secrets:\n  TOK: %s\n", encrypt(t, "V", testMaster)))
		before := readFile(t, manifestPath(source))

		runRotate(t, RotateOptions{Source: source, NewMasterKey: testNewMaster}, true)

		assert.Equal(t, readFile(t, manifestPath(source)), before)
	})

	t.Run("MidRunFailureNoHalfRotate", func(t *testing.T) {
		source := newSource(t)
		target := t.TempDir()
		good := filepath.Join(target, "good")
		bad := filepath.Join(target, "bad")
		write(t, rhymingSource(source, good), encrypt(t, "GOOD\n", testMaster))
		write(t, rhymingSource(source, bad), encrypt(t, "BAD\n", otherMaster))
		writeManifest(t, source, fmt.Sprintf(
			"seeds:\n  %s:\n    secret: true\n  %s:\n    secret: true\n",
			good, bad,
		))
		goodBefore := readFile(t, rhymingSource(source, good))

		rotate(t, source, true)

		assert.Equal(t, readFile(t, rhymingSource(source, good)), goodBefore)
		decryptsUnder(t, readFile(t, rhymingSource(source, good)), testMaster, "GOOD\n")
	})

	t.Run("ReadOnlyTargetFailsClosedBeforeWrite", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root bypasses directory write permissions")
		}

		source := newSource(t)
		dest := filepath.Join(t.TempDir(), "locked", "id_key")
		write(t, rhymingSource(source, dest), encrypt(t, "MIRROR\n", testMaster))
		writeManifest(t, source, fmt.Sprintf(
			"secrets:\n  TOK: %s\nseeds:\n  %s:\n    secret: true\n",
			encrypt(t, "V", testMaster), dest,
		))

		lockedDir := filepath.Dir(rhymingSource(source, dest))
		assert.NilError(t, os.Chmod(lockedDir, 0o500))
		t.Cleanup(func() { os.Chmod(lockedDir, 0o755) })
		before := readFile(t, manifestPath(source))

		rotate(t, source, true)

		assert.Equal(t, readFile(t, manifestPath(source)), before)
		decryptsUnder(t, readFile(t, rhymingSource(source, dest)), testMaster, "MIRROR\n")
	})

	t.Run("SymlinkTargetFailsClosedBeforeWrite", func(t *testing.T) {
		source := newSource(t)
		dir := t.TempDir()
		plain := filepath.Join(dir, "a.enc")
		real := filepath.Join(dir, "z.enc")
		link := filepath.Join(dir, "z.link")
		write(t, plain, encrypt(t, "AVAL", testMaster))
		write(t, real, encrypt(t, "ZVAL", testMaster))
		assert.NilError(t, os.Symlink(real, link))
		writeManifest(t, source, fmt.Sprintf("secrets:\n  A: file:%s\n  Z: file:%s\n", plain, link))

		rotate(t, source, true)

		decryptsUnder(t, readFile(t, plain), testMaster, "AVAL")
		decryptsUnder(t, readFile(t, real), testMaster, "ZVAL")
	})

	t.Run("OldEqualsNewReEncrypts", func(t *testing.T) {
		source := newSource(t)
		original := encrypt(t, "V", testMaster)
		writeManifest(t, source, fmt.Sprintf("secrets:\n  TOK: %s\n", original))

		runRotate(t, RotateOptions{Source: source, MasterKey: testMaster, NewMasterKey: testMaster}, false)

		manifest := loadManifest(t, source)
		assert.Assert(t, manifest.Secrets["TOK"] != original)
		decryptsUnder(t, manifest.Secrets["TOK"], testMaster, "V")
	})

	t.Run("SelfReferencingFileRefFailsWithoutLooping", func(t *testing.T) {
		source := newSource(t)
		loop := filepath.Join(t.TempDir(), "loop.enc")
		write(t, loop, "file:"+loop)
		writeManifest(t, source, fmt.Sprintf("secrets:\n  TOK: file:%s\n", loop))

		rotate(t, source, true)
	})
}
