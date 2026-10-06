package seed

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseManifest(t *testing.T) {
	rejected := []struct {
		name string
		yaml string
		want string
	}{
		{"UnknownVersionRejected", "version: v2\n", "unsupported manifest version"},
		{"MissingVersionRejected", "seeds: {}\n", "unsupported manifest version"},
		{"CopyOnlyEntryRejected", "version: v1\nseeds:\n  /tmp/x:\n    op: copy\n", "copy-only entry is not allowed"},
		{"EmptyEntryRejected", "version: v1\nseeds:\n  /tmp/x: {}\n", "copy-only entry is not allowed"},
		{"SecretValueInvalidRejected", "version: v1\nsecrets:\n  TOKEN: plainnodollar\n", `secret "TOKEN": expected ciphertext or file: ref`},
		{"CommentOnLineinfileRejected", "version: v1\nseeds:\n  /tmp/x:\n    op: lineinfile\n    comment: \"//\"\n    content: \"x\\n\"\n", "comment is only valid with op: block"},
		{"UnknownOpRejected", "version: v1\nseeds:\n  /tmp/x:\n    op: smash\n", `unknown op "smash"`},
		{"CommentOnNonBlockRejected", "version: v1\nseeds:\n  /tmp/x:\n    op: append\n    comment: \"//\"\n    content: \"x\\n\"\n", "comment is only valid with op: block"},
	}

	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(tt.yaml))
			assert.ErrorContains(t, err, tt.want)
		})
	}

	accepted := []struct {
		name string
		yaml string
		op   Op
	}{
		{"BehaviorEntryAccepted", "version: v1\nseeds:\n  /tmp/x:\n    secret: true\n", OpCopy},
		{"BlockOpAccepted", "version: v1\nseeds:\n  /tmp/x:\n    op: block\n    content: \"hi\\n\"\n", OpBlock},
		{"LineinfileOpAccepted", "version: v1\nseeds:\n  /tmp/x:\n    op: lineinfile\n    content: \"FOO=1\\n\"\n", OpLineInfile},
	}

	for _, tt := range accepted {
		t.Run(tt.name, func(t *testing.T) {
			manifest, err := ParseManifest([]byte(tt.yaml))
			assert.NilError(t, err)
			assert.Equal(t, manifest.Seeds["/tmp/x"].Op, tt.op)
		})
	}

	t.Run("SecretValueFileRefAccepted", func(t *testing.T) {
		manifest, err := ParseManifest([]byte("version: v1\nsecrets:\n  TOKEN: file:/run/secrets/token\n"))
		assert.NilError(t, err)
		assert.Equal(t, manifest.Secrets["TOKEN"], "file:/run/secrets/token")
	})

	t.Run("InlineContentEntryAccepted", func(t *testing.T) {
		manifest, err := ParseManifest([]byte("version: v1\nseeds:\n  /tmp/x:\n    content: \"hi\\n\"\n"))
		assert.NilError(t, err)
		assert.Equal(t, *manifest.Seeds["/tmp/x"].Content, "hi\n")
		assert.Equal(t, manifest.Seeds["/tmp/x"].Op, OpCopy)
	})
}
