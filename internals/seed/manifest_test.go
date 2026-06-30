package seed

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseManifest(t *testing.T) {
	t.Run("UnknownVersionRejected", func(t *testing.T) {
		_, err := ParseManifest([]byte("version: v2\n"))
		assert.ErrorContains(t, err, "unsupported manifest version")
	})

	t.Run("MissingVersionRejected", func(t *testing.T) {
		_, err := ParseManifest([]byte("seeds: {}\n"))
		assert.ErrorContains(t, err, "unsupported manifest version")
	})

	t.Run("CopyOnlyEntryRejected", func(t *testing.T) {
		_, err := ParseManifest([]byte("version: v1\nseeds:\n  /tmp/x:\n    op: copy\n"))
		assert.ErrorContains(t, err, "copy-only entry is not allowed")
	})

	t.Run("EmptyEntryRejected", func(t *testing.T) {
		_, err := ParseManifest([]byte("version: v1\nseeds:\n  /tmp/x: {}\n"))
		assert.ErrorContains(t, err, "copy-only entry is not allowed")
	})

	t.Run("SecretValueInvalidRejected", func(t *testing.T) {
		_, err := ParseManifest([]byte("version: v1\nsecrets:\n  TOKEN: plainnodollar\n"))
		assert.ErrorContains(t, err, `secret "TOKEN": expected ciphertext or file: ref`)
	})

	t.Run("SecretValueFileRefAccepted", func(t *testing.T) {
		manifest, err := ParseManifest([]byte("version: v1\nsecrets:\n  TOKEN: file:/run/secrets/token\n"))
		assert.NilError(t, err)
		assert.Equal(t, manifest.Secrets["TOKEN"], "file:/run/secrets/token")
	})

	t.Run("BehaviorEntryAccepted", func(t *testing.T) {
		manifest, err := ParseManifest([]byte("version: v1\nseeds:\n  /tmp/x:\n    secret: true\n"))
		assert.NilError(t, err)
		assert.Equal(t, manifest.Seeds["/tmp/x"].Op, OpCopy)
	})

	t.Run("InlineContentEntryAccepted", func(t *testing.T) {
		manifest, err := ParseManifest([]byte("version: v1\nseeds:\n  /tmp/x:\n    content: \"hi\\n\"\n"))
		assert.NilError(t, err)
		assert.Equal(t, *manifest.Seeds["/tmp/x"].Content, "hi\n")
		assert.Equal(t, manifest.Seeds["/tmp/x"].Op, OpCopy)
	})

	t.Run("BlockOpAccepted", func(t *testing.T) {
		manifest, err := ParseManifest([]byte("version: v1\nseeds:\n  /tmp/x:\n    op: block\n    content: \"hi\\n\"\n"))
		assert.NilError(t, err)
		assert.Equal(t, manifest.Seeds["/tmp/x"].Op, OpBlock)
	})

	t.Run("UnknownOpRejected", func(t *testing.T) {
		_, err := ParseManifest([]byte("version: v1\nseeds:\n  /tmp/x:\n    op: smash\n"))
		assert.ErrorContains(t, err, `unknown op "smash"`)
	})

	t.Run("CommentOnNonBlockRejected", func(t *testing.T) {
		_, err := ParseManifest([]byte("version: v1\nseeds:\n  /tmp/x:\n    op: append\n    comment: \"//\"\n    content: \"x\\n\"\n"))
		assert.ErrorContains(t, err, "comment is only valid with op: block")
	})
}
