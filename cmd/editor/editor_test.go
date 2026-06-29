package editor

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestPersistentPreRunEBlocksSSH(t *testing.T) {
	t.Setenv("SSH_CONNECTION", "1.2.3.4 51000 5.6.7.8 22")

	err := EditorCmd.PersistentPreRunE(EditorCmd, nil)
	assert.ErrorContains(t, err, "SSH")
}

func TestParseSelection(t *testing.T) {
	t.Run("SinglePositionIsEmptyRange", func(t *testing.T) {
		got, err := parseSelection("12:5")
		assert.NilError(t, err)

		assert.Equal(t, got.Start.Line, 11)
		assert.Equal(t, got.Start.Character, 4)
		assert.Equal(t, got.End, got.Start)
	})

	t.Run("Range", func(t *testing.T) {
		got, err := parseSelection("1:1-3:8")
		assert.NilError(t, err)

		assert.Equal(t, got.Start.Line, 0)
		assert.Equal(t, got.Start.Character, 0)
		assert.Equal(t, got.End.Line, 2)
		assert.Equal(t, got.End.Character, 7)
	})

	t.Run("Invalid", func(t *testing.T) {
		for _, bad := range []string{"", "5", "abc:1", "1:0", "0:1", "1:2-bogus"} {
			_, err := parseSelection(bad)
			assert.ErrorContains(t, err, "invalid selection", "input %q should fail", bad)
		}
	})
}
