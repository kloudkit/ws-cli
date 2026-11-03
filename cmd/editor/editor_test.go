package editor

import (
	"path/filepath"
	"strings"
	"testing"

	editoripc "github.com/kloudkit/ws-cli/internals/editor"
	"gotest.tools/v3/assert"
)

func TestResolvePath(t *testing.T) {
	got, err := resolvePath("Dockerfile")
	assert.NilError(t, err)
	assert.Assert(t, filepath.IsAbs(got) && filepath.Base(got) == "Dockerfile", got)

	for _, kept := range []string{"/etc/hosts", "vscode-remote://editor.example.test/etc/hosts"} {
		got, err := resolvePath(kept)
		assert.NilError(t, err)
		assert.Equal(t, got, kept)
	}
}

func TestNotifyValidation(t *testing.T) {
	for payload, want := range map[string]string{
		"{not json":                     "invalid JSON on stdin",
		`{"detail": "no message here"}`: `requires a JSON payload with a "message"`,
	} {
		notifyCmd.SetIn(strings.NewReader(payload))
		assert.ErrorContains(t, notifyCmd.RunE(notifyCmd, nil), want)
	}
}

func TestPersistentPreRunEBlocksSSH(t *testing.T) {
	t.Setenv("SSH_CONNECTION", "1.2.3.4 51000 5.6.7.8 22")

	assert.ErrorContains(t, EditorCmd.PersistentPreRunE(EditorCmd, nil), "SSH")
}

func TestParseSelection(t *testing.T) {
	got, err := parseSelection("12:5")
	assert.NilError(t, err)
	assert.DeepEqual(t, *got, editoripc.Range{Start: editoripc.Position{Line: 11, Character: 4}, End: editoripc.Position{Line: 11, Character: 4}})

	got, err = parseSelection("1:1-3:8")
	assert.NilError(t, err)
	assert.DeepEqual(t, *got, editoripc.Range{End: editoripc.Position{Line: 2, Character: 7}})

	for _, bad := range []string{"", "5", "abc:1", "1:0", "0:1", "1:2-bogus"} {
		_, err := parseSelection(bad)
		assert.ErrorContains(t, err, "invalid selection", bad)
	}
}
