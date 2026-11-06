package editor

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/kloudkit/ws-cli/cmd/clitest"
	"github.com/kloudkit/ws-cli/internals/config"
	editoripc "github.com/kloudkit/ws-cli/internals/editor"
	"github.com/kloudkit/ws-cli/internals/tempfile"
	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
)

func TestMain(m *testing.M) {
	reapLater = func(string) error { return errors.New("the reaper does not run in tests") }

	os.Exit(m.Run())
}

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
	downloadWorkspace(t, "")

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

const waiting = "  Waiting for you to confirm the download in the editor…\n"

func runEditor(t *testing.T, stdin string, args ...string) (stdout, stderr string, code int, err error) {
	t.Helper()

	original := osExit
	osExit = func(c int) { code = c }
	t.Cleanup(func() { osExit = original })

	testutil.ResetCommandFlags(EditorCmd)

	var out, errOut bytes.Buffer
	EditorCmd.SetIn(strings.NewReader(stdin))
	EditorCmd.SetOut(&out)
	EditorCmd.SetErr(&errOut)
	EditorCmd.SetArgs(args)
	err = EditorCmd.Execute()

	return testutil.StripAnsi(out.String()), testutil.StripAnsi(errOut.String()), code, err
}

type fakeEditor struct {
	paths []string
}

func editorAnswering(t *testing.T, body string) *fakeEditor {
	t.Helper()

	editor := &fakeEditor{}
	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var envelope struct {
			Path string `json:"path"`
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &envelope)
		editor.paths = append(editor.paths, envelope.Path)
		_, _ = w.Write([]byte(body))
	}))

	return editor
}

func downloadWorkspace(t *testing.T, lock string) string {
	t.Helper()

	testutil.FakeWorkspace(t)
	t.Setenv("WS_EDITOR_DISABLE_FILE_DOWNLOADS", lock)

	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)

	return tmp
}

func handedOff(t *testing.T) *[]string {
	t.Helper()

	zips := &[]string{}
	original := reapLater
	reapLater = func(zip string) error {
		*zips = append(*zips, zip)
		return nil
	}
	t.Cleanup(func() { reapLater = original })

	return zips
}

func leftZips(t *testing.T, tmp string) []string {
	t.Helper()

	left, err := filepath.Glob(filepath.Join(tmp, "ws-cli-download-*", "*"))
	assert.NilError(t, err)

	return left
}

func leftFolders(t *testing.T, tmp string) []string {
	t.Helper()

	left, err := filepath.Glob(filepath.Join(tmp, "ws-cli-download-*"))
	assert.NilError(t, err)

	return left
}

func pretendTerminal(t *testing.T) {
	t.Helper()

	original := isTerminal
	isTerminal = func(io.Reader) bool { return true }
	t.Cleanup(func() { isTerminal = original })
}

func buildFolder(t *testing.T) string {
	t.Helper()

	folder := filepath.Join(t.TempDir(), "build")
	assert.NilError(t, os.MkdirAll(filepath.Join(folder, "sub"), 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(folder, "a.txt"), []byte("alpha"), 0o600))
	assert.NilError(t, os.WriteFile(filepath.Join(folder, "sub", "b.txt"), []byte("beta"), 0o600))
	assert.NilError(t, os.Symlink("a.txt", filepath.Join(folder, "link")))

	return folder
}

func zipContents(t *testing.T, path string) map[string]string {
	t.Helper()

	archive, err := zip.OpenReader(path)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = archive.Close() })

	contents := map[string]string{}
	for _, entry := range archive.File {
		reader, err := entry.Open()
		assert.NilError(t, err)
		body, err := io.ReadAll(reader)
		assert.NilError(t, err)
		_ = reader.Close()
		contents[entry.Name] = string(body)
	}

	return contents
}

func TestDownloadZipsAFolder(t *testing.T) {
	tmp := downloadWorkspace(t, "")
	editor := editorAnswering(t, `{"downloaded":true}`)
	reaped := handedOff(t)
	folder := buildFolder(t)

	stdout, stderr, code, err := runEditor(t, "", "download", "--zip", folder)

	assert.NilError(t, err)
	assert.Equal(t, code, 0)
	assert.Equal(t, len(editor.paths), 1)

	zipped := editor.paths[0]
	assert.Equal(t, filepath.Base(zipped), "build.zip")
	assert.Equal(t, filepath.Dir(filepath.Dir(zipped)), tmp)
	assert.Assert(t, strings.HasPrefix(filepath.Base(filepath.Dir(zipped)), "ws-cli-download-"), zipped)
	assert.Equal(t, stderr, "✓ Zipped build into "+zipped+"\n"+waiting)
	assert.Assert(t, strings.Contains(stdout, "Download started") && strings.Contains(stdout, zipped), stdout)
	assert.DeepEqual(t, *reaped, []string{zipped})
	assert.DeepEqual(t, zipContents(t, zipped), map[string]string{
		"build/":          "",
		"build/a.txt":     "alpha",
		"build/link":      "a.txt",
		"build/sub/":      "",
		"build/sub/b.txt": "beta",
	})
}

func TestDownloadRemovesTheZipWhenTheDownloadDoesNotStart(t *testing.T) {
	for name, answer := range map[string]http.HandlerFunc{
		"declined": func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"downloaded":false}`)) },
		"refused": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`"File downloads are disabled."`))
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmp := downloadWorkspace(t, "")
			testutil.StartIPC(t, answer)
			reaped := handedOff(t)

			_, err := clitest.Run(EditorCmd, "", "download", "--zip", buildFolder(t))

			assert.Assert(t, err != nil)
			assert.Equal(t, len(leftFolders(t, tmp)), 0)
			assert.Equal(t, len(*reaped), 0)
		})
	}
}

func TestDownloadRemovesTheZipWhenTheWaitRunsOut(t *testing.T) {
	tmp := downloadWorkspace(t, "")
	original := editoripc.DownloadWait
	editoripc.DownloadWait = 200 * time.Millisecond
	t.Cleanup(func() { editoripc.DownloadWait = original })
	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))

	_, err := clitest.Run(EditorCmd, "", "download", "--zip", buildFolder(t))

	assert.Error(t, err, "the workspace editor did not answer within 200ms")
	assert.Equal(t, len(leftFolders(t, tmp)), 0)
}

func TestDownloadAsksBeforeZippingInATerminal(t *testing.T) {
	for answer, wantSent := range map[string]bool{"y\n": true, "YES\n": true, "n\n": false, "\n": false} {
		t.Run(strings.TrimSpace(answer), func(t *testing.T) {
			tmp := downloadWorkspace(t, "")
			editor := editorAnswering(t, `{"downloaded":true}`)
			reaped := handedOff(t)
			pretendTerminal(t)
			folder := buildFolder(t)

			_, stderr, code, err := runEditor(t, answer, "download", folder)

			assert.NilError(t, err)
			assert.Assert(t, strings.HasPrefix(stderr, "Zip build into build.zip and download it? [y/N] "), stderr)

			if wantSent {
				assert.Equal(t, code, 0)
				assert.DeepEqual(t, editor.paths, leftZips(t, tmp))
				assert.DeepEqual(t, *reaped, editor.paths)
			} else {
				assert.Equal(t, code, 1)
				assert.Equal(t, len(editor.paths), 0)
				assert.Equal(t, len(leftFolders(t, tmp)), 0)
			}
		})
	}
}

func TestDownloadRefusesAFolderWithoutATerminal(t *testing.T) {
	tmp := downloadWorkspace(t, "")
	editor := editorAnswering(t, `{"downloaded":true}`)
	folder := buildFolder(t)

	stdout, stderr, code, err := runEditor(t, "y\n", "download", folder)

	assert.NilError(t, err)
	assert.Equal(t, code, 1)
	assert.Equal(t, stdout, "")
	assert.Equal(t, stderr, "⚠ "+folder+" is a folder; pass --zip to zip it and download the zip\n")
	assert.Equal(t, len(editor.paths), 0)

	left, _ := filepath.Glob(filepath.Join(tmp, "*"))
	assert.Equal(t, len(left), 0)
}

func TestDownloadSweepsOldZips(t *testing.T) {
	tmp := downloadWorkspace(t, "")
	editorAnswering(t, `{"downloaded":true}`)

	old, fresh := filepath.Join(tmp, "ws-cli-download-1"), filepath.Join(tmp, "ws-cli-download-2")
	for _, folder := range []string{old, fresh} {
		assert.NilError(t, os.Mkdir(folder, 0o700))
		assert.NilError(t, os.WriteFile(filepath.Join(folder, "build.zip"), nil, 0o600))
	}
	stale := time.Now().Add(-2 * time.Hour)
	assert.NilError(t, os.Chtimes(old, stale, stale))

	file := filepath.Join(t.TempDir(), "big.iso")
	assert.NilError(t, os.WriteFile(file, nil, 0o600))

	_, err := clitest.Run(EditorCmd, "", "download", file)

	assert.NilError(t, err)
	assert.DeepEqual(t, leftFolders(t, tmp), []string{fresh})
}

func TestDownloadNamesTheZip(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{nil, "build.zip"},
		{[]string{"--name", "release"}, "release.zip"},
		{[]string{"--name", "release.zip"}, "release.zip"},
		{[]string{"--name", "Release.ZIP"}, "Release.ZIP"},
	} {
		t.Run(c.want, func(t *testing.T) {
			downloadWorkspace(t, "")
			editor := editorAnswering(t, `{"downloaded":true}`)
			handedOff(t)

			_, err := clitest.Run(EditorCmd, "", append([]string{"download", "--zip", buildFolder(t)}, c.args...)...)

			assert.NilError(t, err)
			assert.Equal(t, len(editor.paths), 1)
			assert.Equal(t, filepath.Base(editor.paths[0]), c.want)
		})
	}
}

func TestDownloadNameOnlyAppliesToFolders(t *testing.T) {
	downloadWorkspace(t, "")
	editor := editorAnswering(t, `{"downloaded":true}`)

	file := filepath.Join(t.TempDir(), "big.iso")
	assert.NilError(t, os.WriteFile(file, nil, 0o600))

	_, err := clitest.Run(EditorCmd, "", "download", "--name", "other", file)
	assert.Error(t, err, "not a folder: --name only applies to folders")

	_, err = clitest.Run(EditorCmd, "", "download", "--name", "other", "/dev/zero")
	assert.Error(t, err, "not a regular file: /dev/zero")

	assert.Equal(t, len(editor.paths), 0)
}

func TestDownloadRefusesANameThatIsAPath(t *testing.T) {
	tmp := downloadWorkspace(t, "")
	editor := editorAnswering(t, `{"downloaded":true}`)

	_, err := clitest.Run(EditorCmd, "", "download", "--zip", "--name", "../escape", buildFolder(t))

	assert.ErrorContains(t, err, `"../escape.zip" is not a file name`)
	assert.Equal(t, len(editor.paths), 0)
	assert.Equal(t, len(leftFolders(t, tmp)), 0)
}

func TestDownloadRefusesWhatIsNotARegularFile(t *testing.T) {
	downloadWorkspace(t, "")
	editor := editorAnswering(t, `{"downloaded":true}`)

	dir := t.TempDir()
	fifo, dangling := filepath.Join(dir, "pipe"), filepath.Join(dir, "dangling")
	assert.NilError(t, syscall.Mkfifo(fifo, 0o600))
	assert.NilError(t, os.Symlink(filepath.Join(dir, "absent"), dangling))

	for path, want := range map[string]string{
		"/dev/zero": "not a regular file: /dev/zero",
		fifo:        "not a regular file: " + fifo,
		dangling:    dangling + ": no such file or directory",
	} {
		_, err := clitest.Run(EditorCmd, "", "download", path)
		assert.Error(t, err, want)
	}

	assert.Equal(t, len(editor.paths), 0)
}

func TestDownloadRemovesTheZipWhenZippingFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads any file")
	}

	tmp := downloadWorkspace(t, "")
	editor := editorAnswering(t, `{"downloaded":true}`)
	folder := buildFolder(t)
	assert.NilError(t, os.WriteFile(filepath.Join(folder, "secret"), []byte("x"), 0o000))

	_, err := clitest.Run(EditorCmd, "", "download", "--zip", folder)

	assert.ErrorContains(t, err, "cannot zip "+folder)
	assert.Equal(t, len(editor.paths), 0)
	assert.Equal(t, len(leftFolders(t, tmp)), 0)
}

func TestDownloadWarnsAboutSkippedSpecialFiles(t *testing.T) {
	downloadWorkspace(t, "")
	editor := editorAnswering(t, `{"downloaded":true}`)
	handedOff(t)
	folder := buildFolder(t)
	fifo := filepath.Join(folder, "pipe")
	assert.NilError(t, syscall.Mkfifo(fifo, 0o600))

	_, stderr, code, err := runEditor(t, "", "download", "--zip", folder)

	assert.NilError(t, err)
	assert.Equal(t, code, 0)
	assert.Assert(t, strings.HasPrefix(stderr, "⚠ skipped "+fifo+": not a regular file, folder or link\n"), stderr)
	assert.Equal(t, len(editor.paths), 1)
	_, zipped := zipContents(t, editor.paths[0])["build/pipe"]
	assert.Assert(t, !zipped)
}

func TestDownloadRawWithAZip(t *testing.T) {
	for _, c := range []struct {
		body       string
		downloaded bool
	}{{`{"downloaded":true}`, true}, {`{"downloaded":false}`, false}} {
		t.Run(c.body, func(t *testing.T) {
			tmp := downloadWorkspace(t, "")
			editor := editorAnswering(t, c.body)
			reaped := handedOff(t)

			stdout, _, _, err := runEditor(t, "", "download", "--raw", "--zip", buildFolder(t))

			assert.Equal(t, strings.SplitN(stdout, "\n", 2)[0], c.body)
			if c.downloaded {
				assert.NilError(t, err)
				assert.DeepEqual(t, *reaped, editor.paths)
				assert.DeepEqual(t, leftZips(t, tmp), editor.paths)
			} else {
				assert.Error(t, err, "the download was cancelled or timed out in the editor")
				assert.Equal(t, len(*reaped), 0)
				assert.Equal(t, len(leftFolders(t, tmp)), 0)
			}
		})
	}
}

func TestDownloadRoutesAFolderByTheFolder(t *testing.T) {
	tmp := downloadWorkspace(t, "")
	handedOff(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("VSCODE_IPC_HOOK_CLI", "")

	pipe := filepath.Join(home, "pipe.sock")
	session := filepath.Join(home, ".local", "share", "ws-server", "session.sock")
	assert.NilError(t, os.MkdirAll(filepath.Dir(session), 0o755))

	var routedBy string
	testutil.ServeUnix(t, session, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routedBy = r.URL.Query().Get("filePath")
		_ = json.NewEncoder(w).Encode(map[string]string{"socketPath": pipe})
	}))

	var sent string
	testutil.ServeUnix(t, pipe, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var envelope struct {
			Path string `json:"path"`
		}
		_ = json.NewDecoder(r.Body).Decode(&envelope)
		sent = envelope.Path
		_, _ = w.Write([]byte(`{"downloaded":true}`))
	}))

	folder := buildFolder(t)
	_, err := clitest.Run(EditorCmd, "", "download", "--zip", folder)

	assert.NilError(t, err)
	assert.Equal(t, routedBy, folder)
	assert.Equal(t, filepath.Dir(filepath.Dir(sent)), tmp)
}

func TestReapDownloadRemovesAReadZipAndItsFolder(t *testing.T) {
	tmp := downloadWorkspace(t, "")

	file, err := tempfile.Download.Create("build.zip")
	assert.NilError(t, err)
	_, err = file.WriteString("zip")
	assert.NilError(t, err)
	assert.NilError(t, file.Close())

	info, err := os.Stat(file.Name())
	assert.NilError(t, err)
	assert.NilError(t, os.Chtimes(file.Name(), info.ModTime().Add(time.Second), info.ModTime()))

	_, err = clitest.Run(EditorCmd, "", "reap-download", file.Name())

	assert.NilError(t, err)
	assert.Equal(t, len(leftFolders(t, tmp)), 0)
}

func TestDownloadTrapRemovesTheZipOnSIGINT(t *testing.T) {
	tmp := downloadWorkspace(t, "")
	folder := buildFolder(t)

	socket := filepath.Join(t.TempDir(), "pipe.sock")
	arrived := make(chan struct{}, 1)
	testutil.ServeUnix(t, socket, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		arrived <- struct{}{}
		<-r.Context().Done()
	}))

	child := exec.Command(os.Args[0], "-test.run=^TestDownloadHelperProcess$")
	child.Env = append(os.Environ(),
		helperArgsEnv+"=download\x1f--zip\x1f"+folder,
		helperManifestEnv+"="+config.DefaultManifestPath,
		"VSCODE_IPC_HOOK_CLI="+socket,
	)
	assert.NilError(t, child.Start())

	select {
	case <-arrived:
	case <-time.After(10 * time.Second):
		_ = child.Process.Kill()
		t.Fatal("the download request never arrived")
	}

	assert.NilError(t, child.Process.Signal(os.Interrupt))

	var exitErr *exec.ExitError
	assert.Assert(t, errors.As(child.Wait(), &exitErr))
	assert.Equal(t, exitErr.ExitCode(), 130)

	left, _ := filepath.Glob(filepath.Join(tmp, "*"))
	assert.Equal(t, len(left), 0)
}

const (
	helperArgsEnv     = "EDITOR_TEST_HELPER_ARGS"
	helperManifestEnv = "EDITOR_TEST_HELPER_MANIFEST"
)

func TestDownloadHelperProcess(t *testing.T) {
	args := os.Getenv(helperArgsEnv)
	if args == "" {
		return
	}

	config.DefaultManifestPath = os.Getenv(helperManifestEnv)
	EditorCmd.SetArgs(strings.Split(args, "\x1f"))
	_ = EditorCmd.Execute()
	os.Exit(0)
}

func TestDownloadRefusedWhenFileDownloadsDisabled(t *testing.T) {
	for _, value := range []string{"1", "true", "yes", "on", "TRUE", "Yes"} {
		t.Run(value, func(t *testing.T) {
			downloadWorkspace(t, value)
			editor := editorAnswering(t, `{"downloaded":true}`)

			file := filepath.Join(t.TempDir(), "big.iso")
			assert.NilError(t, os.WriteFile(file, nil, 0o600))

			for _, target := range []string{file, buildFolder(t), filepath.Join(t.TempDir(), "absent.iso")} {
				_, err := clitest.Run(EditorCmd, "", "download", "--zip", target)

				assert.Error(t, err, "file downloads are disabled in this workspace (WS_EDITOR_DISABLE_FILE_DOWNLOADS)", target)
			}

			assert.Equal(t, len(editor.paths), 0)
		})
	}
}

func TestDownloadSendsAFile(t *testing.T) {
	for _, value := range []string{"", "false", "off"} {
		t.Run("lock="+value, func(t *testing.T) {
			downloadWorkspace(t, value)
			editor := editorAnswering(t, `{"downloaded":true}`)

			file := filepath.Join(t.TempDir(), "big.iso")
			assert.NilError(t, os.WriteFile(file, nil, 0o600))

			out, err := clitest.Run(EditorCmd, "", "download", file)

			assert.NilError(t, err)
			assert.Assert(t, strings.Contains(out, "Download started"), out)
			assert.DeepEqual(t, editor.paths, []string{file})
		})
	}
}

func TestDownloadAnsweredFalse(t *testing.T) {
	downloadWorkspace(t, "")
	editorAnswering(t, `{"downloaded":false}`)

	file := filepath.Join(t.TempDir(), "big.iso")
	assert.NilError(t, os.WriteFile(file, nil, 0o600))

	_, err := clitest.Run(EditorCmd, "", "download", file)

	assert.Error(t, err, "the download was cancelled or timed out in the editor")
}
