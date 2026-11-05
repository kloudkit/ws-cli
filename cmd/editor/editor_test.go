package editor

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/cmd/clitest"
	editoripc "github.com/kloudkit/ws-cli/internals/editor"
	"github.com/kloudkit/ws-cli/internals/testutil"
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

func TestPathCommandsRejectMissingPath(t *testing.T) {
	testutil.FakeWorkspace(t)

	missing := filepath.Join(t.TempDir(), "absent.iso")

	for _, command := range []string{"download", "reveal"} {
		_, err := clitest.Run(EditorCmd, "", command, missing)
		assert.ErrorContains(t, err, missing+": no such file or directory")
	}
}

func TestTerminalNeedsDirectory(t *testing.T) {
	testutil.FakeWorkspace(t)

	file := filepath.Join(t.TempDir(), "file.txt")
	assert.NilError(t, os.WriteFile(file, nil, 0o600))

	_, err := clitest.Run(EditorCmd, "", "terminal", "--cwd", file)
	assert.ErrorContains(t, err, file+": not a directory")
}

func TestOpenModalExcludesTabFlags(t *testing.T) {
	testutil.FakeWorkspace(t)

	for _, other := range []string{"--new-window", "--reuse-window", "--preview"} {
		_, err := clitest.Run(EditorCmd, "", "open", "README.md", "--modal", other)
		assert.ErrorContains(t, err, "none of the others can be")
	}
}

func runEditor(t *testing.T, args ...string) (stdout, stderr string, code int, err error) {
	t.Helper()

	original := osExit
	osExit = func(c int) { code = c }
	t.Cleanup(func() { osExit = original })

	testutil.ResetCommandFlags(EditorCmd)

	var out, errOut bytes.Buffer
	EditorCmd.SetOut(&out)
	EditorCmd.SetErr(&errOut)
	EditorCmd.SetArgs(args)
	err = EditorCmd.Execute()

	return testutil.StripAnsi(out.String()), testutil.StripAnsi(errOut.String()), code, err
}

func editorAnswering(t *testing.T, body string) *int {
	t.Helper()

	requests := 0
	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(body))
	}))

	return &requests
}

func TestDownloadRefusesFolder(t *testing.T) {
	testutil.FakeWorkspace(t)
	requests := editorAnswering(t, `{"downloaded":true}`)

	folder := filepath.Join(t.TempDir(), "build")
	assert.NilError(t, os.Mkdir(folder, 0o755))

	stdout, stderr, code, err := runEditor(t, "download", folder)

	assert.NilError(t, err)
	assert.Equal(t, code, 1)
	assert.Equal(t, stdout, "")
	assert.Equal(t, stderr, "⚠ "+folder+" is a folder; archive it first: tar -czf build.tgz build\n")
	assert.Equal(t, *requests, 0)
}

func TestDownloadRefusedWhenFileDownloadsDisabled(t *testing.T) {
	for _, value := range []string{"1", "true", "yes", "on", "TRUE", "Yes"} {
		t.Run(value, func(t *testing.T) {
			testutil.FakeWorkspace(t)
			requests := editorAnswering(t, `{"downloaded":true}`)
			t.Setenv("WS_EDITOR_DISABLE_FILE_DOWNLOADS", value)

			file := filepath.Join(t.TempDir(), "big.iso")
			assert.NilError(t, os.WriteFile(file, nil, 0o600))

			_, err := clitest.Run(EditorCmd, "", "download", file)

			assert.Error(t, err, "file downloads are disabled in this workspace (WS_EDITOR_DISABLE_FILE_DOWNLOADS)")
			assert.Equal(t, *requests, 0)
		})
	}
}

func TestDownloadSendsAFile(t *testing.T) {
	for _, value := range []string{"", "false", "off"} {
		t.Run("lock="+value, func(t *testing.T) {
			testutil.FakeWorkspace(t)
			requests := editorAnswering(t, `{"downloaded":true}`)
			t.Setenv("WS_EDITOR_DISABLE_FILE_DOWNLOADS", value)

			file := filepath.Join(t.TempDir(), "big.iso")
			assert.NilError(t, os.WriteFile(file, nil, 0o600))

			out, err := clitest.Run(EditorCmd, "", "download", file)

			assert.NilError(t, err)
			assert.Assert(t, strings.Contains(out, "Download started"), out)
			assert.Equal(t, *requests, 1)
		})
	}
}
