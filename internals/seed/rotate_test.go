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

func rotate(t *testing.T, opts RotateOptions) string {
	t.Helper()
	var buffer bytes.Buffer
	opts.Out = &buffer
	assert.NilError(t, Rotate(opts))
	return buffer.String()
}

func rotateErr(t *testing.T, opts RotateOptions) {
	t.Helper()
	var buffer bytes.Buffer
	opts.Out = &buffer
	assert.Assert(t, Rotate(opts) != nil)
}

func decrypts(t *testing.T, ciphertext, key, want string) {
	t.Helper()
	master, err := secrets.ResolveMasterKey(key)
	assert.NilError(t, err)
	plain, err := secrets.Decrypt(secrets.NormalizeEncrypted(ciphertext), master)
	assert.NilError(t, err)
	assert.Equal(t, string(plain), want)
}

func failsToDecrypt(t *testing.T, ciphertext, key string) {
	t.Helper()
	master, err := secrets.ResolveMasterKey(key)
	assert.NilError(t, err)
	_, err = secrets.Decrypt(secrets.NormalizeEncrypted(ciphertext), master)
	assert.Assert(t, err != nil)
}

func TestRotate(t *testing.T) {
	t.Run("NewMasterRequired", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		writeManifest(t, source, fmt.Sprintf("secrets:\n  TOK: %s\n", encrypt(t, "V", testMaster)))

		rotateErr(t, RotateOptions{Source: source, MasterKey: testMaster})
	})

	t.Run("RotatesInlineSecretsMap", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		writeManifest(t, source, fmt.Sprintf("secrets:\n  TOK: %s\n", encrypt(t, "TOKENVAL", testMaster)))

		rotate(t, RotateOptions{Source: source, MasterKey: testMaster, NewMasterKey: testNewMaster})

		manifest, err := LoadManifest(ManifestPath(source))
		assert.NilError(t, err)
		decrypts(t, manifest.Secrets["TOK"], testNewMaster, "TOKENVAL")
		failsToDecrypt(t, manifest.Secrets["TOK"], testMaster)
	})

	t.Run("RotatesFileRef", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		keyFile := filepath.Join(t.TempDir(), "tok.enc")
		write(t, keyFile, encrypt(t, "FILEVAL", testMaster))
		writeManifest(t, source, fmt.Sprintf("secrets:\n  TOK: file:%s\n", keyFile))

		rotate(t, RotateOptions{Source: source, MasterKey: testMaster, NewMasterKey: testNewMaster})

		manifest, err := LoadManifest(ManifestPath(source))
		assert.NilError(t, err)
		assert.Equal(t, manifest.Secrets["TOK"], "file:"+keyFile)
		decrypts(t, readFile(t, keyFile), testNewMaster, "FILEVAL")
		failsToDecrypt(t, readFile(t, keyFile), testMaster)
	})

	t.Run("RotatesSecretTrueMirror", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "id_key")
		write(t, rhyming(source, dest), encrypt(t, "PRIVKEY\n", testMaster))
		writeManifest(t, source, fmt.Sprintf("seeds:\n  %s:\n    secret: true\n", dest))

		rotate(t, RotateOptions{Source: source, MasterKey: testMaster, NewMasterKey: testNewMaster})

		mirror := readFile(t, rhyming(source, dest))
		decrypts(t, mirror, testNewMaster, "PRIVKEY\n")
		failsToDecrypt(t, mirror, testMaster)
	})

	t.Run("RotatesSecretTrueInlineContent", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "id_key")
		writeManifest(t, source, fmt.Sprintf(
			"seeds:\n  %s:\n    secret: true\n    content: \"%s\"\n",
			dest, encrypt(t, "INLINEKEY\n", testMaster),
		))

		rotate(t, RotateOptions{Source: source, MasterKey: testMaster, NewMasterKey: testNewMaster})

		manifest, err := LoadManifest(ManifestPath(source))
		assert.NilError(t, err)
		decrypts(t, *manifest.Seeds[dest].Content, testNewMaster, "INLINEKEY\n")
		failsToDecrypt(t, *manifest.Seeds[dest].Content, testMaster)
	})

	t.Run("PreservesCommentsAndKeyOrder", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		manifest := fmt.Sprintf(
			"# managed by ws\nversion: v1\nsecrets:\n  # my token\n  TOK: %s\nseeds:\n  /tmp/keep:\n    content: \"plain\\n\"\n",
			encrypt(t, "V", testMaster),
		)
		write(t, ManifestPath(source), manifest)

		rotate(t, RotateOptions{Source: source, MasterKey: testMaster, NewMasterKey: testNewMaster})

		after := readFile(t, ManifestPath(source))
		assert.Assert(t, strings.Contains(after, "# managed by ws"))
		assert.Assert(t, strings.Contains(after, "# my token"))
		assert.Assert(t, strings.Contains(after, "plain"))
		assert.Assert(t, strings.Index(after, "secrets:") < strings.Index(after, "seeds:"))

		parsed, err := ParseManifest([]byte(after))
		assert.NilError(t, err)
		decrypts(t, parsed.Secrets["TOK"], testNewMaster, "V")
	})

	t.Run("WrongOldKeyFailsClosed", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		writeManifest(t, source, fmt.Sprintf("secrets:\n  TOK: %s\n", encrypt(t, "V", testMaster)))
		before := readFile(t, ManifestPath(source))

		rotateErr(t, RotateOptions{
			Source:       source,
			MasterKey:    "a-totally-different-master-key-9999",
			NewMasterKey: testNewMaster,
		})

		assert.Equal(t, readFile(t, ManifestPath(source)), before)
	})

	t.Run("AbsentOldKeyFailsClosed", func(t *testing.T) {
		setEnv(t, t.TempDir())
		t.Setenv("WS_SECRETS_MASTER_KEY", "")
		source := t.TempDir()
		writeManifest(t, source, fmt.Sprintf("secrets:\n  TOK: %s\n", encrypt(t, "V", testMaster)))
		before := readFile(t, ManifestPath(source))

		rotateErr(t, RotateOptions{Source: source, MasterKey: "", NewMasterKey: testNewMaster})

		assert.Equal(t, readFile(t, ManifestPath(source)), before)
	})

	t.Run("MidRunFailureNoHalfRotate", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		good := filepath.Join(target, "good")
		bad := filepath.Join(target, "bad")
		write(t, rhyming(source, good), encrypt(t, "GOOD\n", testMaster))
		write(t, rhyming(source, bad), encrypt(t, "BAD\n", "some-other-master-key-aaaaaaaaaa"))
		writeManifest(t, source, fmt.Sprintf(
			"seeds:\n  %s:\n    secret: true\n  %s:\n    secret: true\n",
			good, bad,
		))
		goodBefore := readFile(t, rhyming(source, good))

		rotateErr(t, RotateOptions{Source: source, MasterKey: testMaster, NewMasterKey: testNewMaster})

		assert.Equal(t, readFile(t, rhyming(source, good)), goodBefore)
		decrypts(t, readFile(t, rhyming(source, good)), testMaster, "GOOD\n")
	})

	t.Run("ReadOnlyTargetFailsClosedBeforeWrite", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root bypasses directory write permissions")
		}

		setEnv(t, t.TempDir())
		source := t.TempDir()
		target := t.TempDir()
		dest := filepath.Join(target, "locked", "id_key")
		write(t, rhyming(source, dest), encrypt(t, "MIRROR\n", testMaster))
		writeManifest(t, source, fmt.Sprintf(
			"secrets:\n  TOK: %s\nseeds:\n  %s:\n    secret: true\n",
			encrypt(t, "V", testMaster), dest,
		))

		lockedDir := filepath.Dir(rhyming(source, dest))
		assert.NilError(t, os.Chmod(lockedDir, 0o500))
		t.Cleanup(func() { os.Chmod(lockedDir, 0o755) })
		before := readFile(t, ManifestPath(source))

		rotateErr(t, RotateOptions{Source: source, MasterKey: testMaster, NewMasterKey: testNewMaster})

		assert.Equal(t, readFile(t, ManifestPath(source)), before)
		decrypts(t, readFile(t, rhyming(source, dest)), testMaster, "MIRROR\n")
	})

	t.Run("OldEqualsNewReEncrypts", func(t *testing.T) {
		setEnv(t, t.TempDir())
		source := t.TempDir()
		original := encrypt(t, "V", testMaster)
		writeManifest(t, source, fmt.Sprintf("secrets:\n  TOK: %s\n", original))

		rotate(t, RotateOptions{Source: source, MasterKey: testMaster, NewMasterKey: testMaster})

		manifest, err := LoadManifest(ManifestPath(source))
		assert.NilError(t, err)
		assert.Assert(t, manifest.Secrets["TOK"] != original)
		decrypts(t, manifest.Secrets["TOK"], testMaster, "V")
	})
}
