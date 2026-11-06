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

	file, err := Download.Create("build")
	assert.NilError(t, err)
	assert.NilError(t, file.Close())

	return file.Name()
}

func TestCreateNamesTheFileForItsKind(t *testing.T) {
	dir := isolate(t)

	path := create(t)

	assert.Equal(t, filepath.Dir(path), dir)
	assert.Assert(t, strings.HasPrefix(filepath.Base(path), "ws-cli-download-build-") && strings.HasSuffix(path, ".zip"), path)
	assert.Assert(t, Download.Owns(path))
}

func TestOwnsOnlyThisUsersRegularFilesOfTheKind(t *testing.T) {
	dir := isolate(t)
	owned := create(t)

	other := filepath.Join(dir, "notes.zip")
	assert.NilError(t, os.WriteFile(other, nil, 0o600))

	nested := filepath.Join(dir, "sub", "ws-cli-download-x.zip")
	assert.NilError(t, os.MkdirAll(filepath.Dir(nested), 0o700))
	assert.NilError(t, os.WriteFile(nested, nil, 0o600))

	link := filepath.Join(dir, "ws-cli-download-link.zip")
	assert.NilError(t, os.Symlink(owned, link))

	folder := filepath.Join(dir, "ws-cli-download-folder.zip")
	assert.NilError(t, os.Mkdir(folder, 0o700))

	for _, path := range []string{other, nested, link, folder, "ws-cli-download-x.zip", filepath.Join(dir, "ws-cli-download-absent.zip")} {
		assert.Assert(t, !Download.Owns(path), path)
		assert.ErrorContains(t, Download.Remove(path), "refusing to remove")
	}

	stub(t, &currentUID, func() int { return os.Getuid() + 1 })
	assert.Assert(t, !Download.Owns(owned))
}

func TestSweepRemovesOnlyOldFilesOfTheKind(t *testing.T) {
	dir := isolate(t)
	old, fresh := create(t), create(t)
	stale := time.Now().Add(-2 * time.Hour)
	assert.NilError(t, os.Chtimes(old, stale, stale))

	other := filepath.Join(dir, "other.zip")
	assert.NilError(t, os.WriteFile(other, nil, 0o600))
	assert.NilError(t, os.Chtimes(other, stale, stale))

	Download.Sweep(time.Hour)

	assert.Assert(t, !exists(old))
	for _, kept := range []string{fresh, other} {
		assert.Assert(t, exists(kept), kept)
	}
}

func TestSweepSkipsLinksAndFolders(t *testing.T) {
	dir := isolate(t)

	target := filepath.Join(dir, "target.txt")
	assert.NilError(t, os.WriteFile(target, nil, 0o600))
	link := filepath.Join(dir, "ws-cli-download-link.zip")
	assert.NilError(t, os.Symlink(target, link))
	folder := filepath.Join(dir, "ws-cli-download-folder.zip")
	assert.NilError(t, os.Mkdir(folder, 0o700))

	Download.Sweep(0)

	for _, kept := range []string{target, link, folder} {
		assert.Assert(t, exists(kept), kept)
	}
}

func TestSweepLeavesAnotherUsersFiles(t *testing.T) {
	isolate(t)
	old := create(t)
	stale := time.Now().Add(-2 * time.Hour)
	assert.NilError(t, os.Chtimes(old, stale, stale))
	stub(t, &currentUID, func() int { return os.Getuid() + 1 })

	Download.Sweep(time.Hour)

	assert.Assert(t, exists(old))
}

func TestTrapRemovesTheFileOnASignal(t *testing.T) {
	for _, received := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
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

			assert.Assert(t, !exists(path))
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

	assert.Assert(t, !exists(path))
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
	assert.Assert(t, !exists(path))
}

func TestReapGivesUpAtTheCap(t *testing.T) {
	isolate(t)
	path := create(t)

	started := time.Now()
	assert.NilError(t, Download.Reap(path, 300*time.Millisecond))

	assert.Assert(t, time.Since(started) >= 300*time.Millisecond)
	assert.Assert(t, !exists(path))
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
