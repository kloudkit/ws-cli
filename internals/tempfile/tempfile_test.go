package tempfile

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func isolate(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)

	return dir
}

func stub[T any](t *testing.T, target *T, value T) {
	t.Helper()

	original := *target
	*target = value
	t.Cleanup(func() { *target = original })
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func create(t *testing.T) string {
	t.Helper()

	file, err := Download.Create("build.zip")
	assert.NilError(t, err)
	assert.NilError(t, file.Close())

	return file.Name()
}

func age(t *testing.T, path string) {
	t.Helper()

	stale := time.Now().Add(-2 * time.Hour)
	assert.NilError(t, os.Chtimes(path, stale, stale))
}

func TestCreatePutsTheFileInItsOwnFolder(t *testing.T) {
	dir := isolate(t)

	path := create(t)

	assert.Equal(t, filepath.Base(path), "build.zip")
	assert.Equal(t, filepath.Dir(filepath.Dir(path)), dir)
	assert.Assert(t, strings.HasPrefix(filepath.Base(filepath.Dir(path)), "ws-cli-download-"), path)
	assert.Assert(t, Download.Owns(path))
}

func TestCreateRefusesAPath(t *testing.T) {
	dir := isolate(t)

	for _, name := range []string{"", ".", "..", "a/b.zip", "/etc/passwd"} {
		_, err := Download.Create(name)
		assert.ErrorContains(t, err, "is not a file name", name)
	}

	left, _ := filepath.Glob(filepath.Join(dir, "*"))
	assert.Equal(t, len(left), 0)
}

func TestARelativeTempDirStillCleansUp(t *testing.T) {
	t.Chdir(t.TempDir())
	assert.NilError(t, os.Mkdir("tmp", 0o700))
	t.Setenv("TMPDIR", "./tmp/../tmp")

	path := create(t)
	assert.Assert(t, filepath.IsAbs(path), path)
	assert.Assert(t, Download.Owns(path))
	assert.NilError(t, Download.Remove(path))

	old := create(t)
	age(t, filepath.Dir(old))
	Download.Sweep(time.Hour)

	left, _ := filepath.Glob("tmp/*")
	assert.Equal(t, len(left), 0)
}

func TestTouchRestartsTheFoldersAge(t *testing.T) {
	isolate(t)
	path := create(t)
	age(t, filepath.Dir(path))

	assert.NilError(t, Download.Touch(path))
	Download.Sweep(time.Hour)

	assert.Assert(t, exists(path))
}

func TestRemoveDeletesTheFileAndItsFolder(t *testing.T) {
	isolate(t)
	path := create(t)

	assert.NilError(t, Download.Remove(path))

	assert.Assert(t, !exists(path))
	assert.Assert(t, !exists(filepath.Dir(path)))
}

func TestOwnsOnlyFilesInThisUsersFoldersOfTheKind(t *testing.T) {
	dir := isolate(t)
	owned := create(t)
	folder := filepath.Dir(owned)

	other := filepath.Join(dir, "other", "build.zip")
	assert.NilError(t, os.MkdirAll(filepath.Dir(other), 0o700))
	assert.NilError(t, os.WriteFile(other, nil, 0o600))

	nested := filepath.Join(folder, "sub", "build.zip")
	assert.NilError(t, os.MkdirAll(filepath.Dir(nested), 0o700))
	assert.NilError(t, os.WriteFile(nested, nil, 0o600))

	outside := filepath.Join(t.TempDir(), "ws-cli-download-x")
	assert.NilError(t, os.Mkdir(outside, 0o700))
	assert.NilError(t, os.WriteFile(filepath.Join(outside, "build.zip"), nil, 0o600))

	linked := filepath.Join(dir, "ws-cli-download-link")
	assert.NilError(t, os.Symlink(outside, linked))

	fileLink := filepath.Join(folder, "link.zip")
	assert.NilError(t, os.Symlink(other, fileLink))

	for _, path := range []string{
		other, nested, filepath.Join(outside, "build.zip"), filepath.Join(linked, "build.zip"), fileLink,
		folder, "build.zip", filepath.Join(folder, "absent.zip"), folder + "/./build.zip",
	} {
		assert.Assert(t, !Download.Owns(path), path)
		assert.ErrorContains(t, Download.Remove(path), "refusing to remove")
	}
	assert.Assert(t, exists(filepath.Join(outside, "build.zip")))

	stub(t, &currentUID, func() int { return os.Getuid() + 1 })
	assert.Assert(t, !Download.Owns(owned))
}

func TestSweepRemovesOnlyOldFoldersOfTheKind(t *testing.T) {
	dir := isolate(t)
	old, fresh := create(t), create(t)
	age(t, filepath.Dir(old))

	other := filepath.Join(dir, "other")
	assert.NilError(t, os.Mkdir(other, 0o700))
	age(t, other)

	crowded := create(t)
	assert.NilError(t, os.WriteFile(filepath.Join(filepath.Dir(crowded), "notes.txt"), nil, 0o600))
	age(t, filepath.Dir(crowded))

	outside := filepath.Join(t.TempDir(), "kept")
	assert.NilError(t, os.Mkdir(outside, 0o700))
	assert.NilError(t, os.WriteFile(filepath.Join(outside, "build.zip"), nil, 0o600))
	linked := filepath.Join(dir, "ws-cli-download-link")
	assert.NilError(t, os.Symlink(outside, linked))

	Download.Sweep(time.Hour)

	assert.Assert(t, !exists(old))
	assert.Assert(t, !exists(filepath.Dir(old)))
	for _, kept := range []string{fresh, other, crowded, linked, filepath.Join(outside, "build.zip")} {
		assert.Assert(t, exists(kept), kept)
	}
}

func TestSweepLeavesAnotherUsersFolders(t *testing.T) {
	isolate(t)
	old := create(t)
	age(t, filepath.Dir(old))
	stub(t, &currentUID, func() int { return os.Getuid() + 1 })

	Download.Sweep(time.Hour)

	assert.Assert(t, exists(old))
}

func TestTrapRemovesTheFileAndFolderOnASignal(t *testing.T) {
	for _, received := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT} {
		t.Run(received.String(), func(t *testing.T) {
			isolate(t)
			path := create(t)

			codes := make(chan int, 1)
			stub(t, &exit, func(code int) { codes <- code })

			release := Download.Trap(path)
			t.Cleanup(release)

			assert.NilError(t, syscall.Kill(os.Getpid(), received))

			select {
			case code := <-codes:
				assert.Equal(t, code, 128+int(received))
			case <-time.After(5 * time.Second):
				t.Fatal("the trap did not fire")
			}

			assert.Assert(t, !exists(filepath.Dir(path)))
		})
	}
}

func TestTrapReleaseKeepsTheFile(t *testing.T) {
	isolate(t)
	path := create(t)

	Download.Trap(path)()

	assert.Assert(t, exists(path))
}

func TestReapRemovesTheFileOnceSomethingHasItOpen(t *testing.T) {
	isolate(t)
	path := create(t)

	reaped := make(chan error, 1)
	go func() { reaped <- Download.Reap(path, time.Minute) }()

	time.Sleep(300 * time.Millisecond)
	assert.Assert(t, exists(path))

	file, err := os.Open(path)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = file.Close() })

	select {
	case err := <-reaped:
		assert.NilError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("the reaper did not notice the open file")
	}

	assert.Assert(t, !exists(filepath.Dir(path)))
}

func TestReapRemovesTheFileOnceItWasRead(t *testing.T) {
	isolate(t)
	path := create(t)
	assert.NilError(t, os.WriteFile(path, []byte("zip"), 0o600))

	info, err := os.Stat(path)
	assert.NilError(t, err)
	assert.NilError(t, os.Chtimes(path, info.ModTime().Add(time.Second), info.ModTime()))

	started := time.Now()
	assert.NilError(t, Download.Reap(path, time.Minute))

	assert.Assert(t, time.Since(started) < 5*time.Second)
	assert.Assert(t, !exists(filepath.Dir(path)))
}

func TestReapGivesUpAtTheCap(t *testing.T) {
	isolate(t)
	path := create(t)

	started := time.Now()
	assert.NilError(t, Download.Reap(path, 300*time.Millisecond))

	assert.Assert(t, time.Since(started) >= 300*time.Millisecond)
	assert.Assert(t, !exists(filepath.Dir(path)))
}

func TestReapRefusesWhatItDoesNotOwn(t *testing.T) {
	dir := isolate(t)
	other := filepath.Join(dir, "keep.zip")
	assert.NilError(t, os.WriteFile(other, nil, 0o600))

	assert.ErrorContains(t, Download.Reap(other, time.Second), "refusing to reap")
	assert.Assert(t, exists(other))
}

func TestDetachStartsTheCommandInItsOwnSession(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "session")
	stub(t, &executable, func() (string, error) { return "/bin/sh", nil })

	assert.NilError(t, Detach("-c", `cut -d" " -f6 /proc/$$/stat > `+out+`.tmp && mv `+out+`.tmp `+out))

	var session string
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if data, err := os.ReadFile(out); err == nil {
			session = strings.TrimSpace(string(data))
			break
		}
	}

	stat, err := os.ReadFile("/proc/self/stat")
	assert.NilError(t, err)
	ours := strings.Fields(string(stat[strings.LastIndexByte(string(stat), ')')+1:]))[3]

	assert.Assert(t, session != "" && session != ours, "child %q, ours %q", session, ours)
}
