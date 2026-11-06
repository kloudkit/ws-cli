package io

import (
	"archive/zip"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"gotest.tools/v3/assert"
)

func zipNames(t *testing.T, path string) []string {
	t.Helper()

	archive, err := zip.OpenReader(path)
	assert.NilError(t, err)
	defer archive.Close()

	var names []string
	for _, entry := range archive.File {
		names = append(names, entry.Name)
	}

	return names
}

func noSkips(t *testing.T) func(string) {
	return func(path string) { t.Errorf("unexpected skip of %s", path) }
}

func TestZipFolderSkipsItsOwnOutput(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "out")
	assert.NilError(t, os.Mkdir(folder, 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(folder, "a.txt"), []byte("alpha"), 0o600))

	target := filepath.Join(folder, "self.zip")
	file, err := os.Create(target)
	assert.NilError(t, err)

	assert.NilError(t, ZipFolder(file, folder, target, noSkips(t)))
	assert.NilError(t, file.Close())

	assert.DeepEqual(t, zipNames(t, target), []string{"out/", "out/a.txt"})
}

func TestZipFolderFollowsASymlinkedFolderAndKeepsTheLinkName(t *testing.T) {
	dir := t.TempDir()
	actual := filepath.Join(dir, "actual")
	assert.NilError(t, os.Mkdir(actual, 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(actual, "a.txt"), []byte("alpha"), 0o600))

	link := filepath.Join(dir, "build")
	assert.NilError(t, os.Symlink(actual, link))

	target := filepath.Join(t.TempDir(), "build.zip")
	file, err := os.Create(target)
	assert.NilError(t, err)

	assert.NilError(t, ZipFolder(file, link, target, noSkips(t)))
	assert.NilError(t, file.Close())

	assert.DeepEqual(t, zipNames(t, target), []string{"build/", "build/a.txt"})
}

func TestZipFolderSkipsSpecialFilesAndKeepsZipping(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "out")
	assert.NilError(t, os.Mkdir(folder, 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(folder, "a.txt"), []byte("alpha"), 0o600))

	fifo := filepath.Join(folder, "pipe")
	assert.NilError(t, syscall.Mkfifo(fifo, 0o600))

	socket := filepath.Join(folder, "sock")
	listener, err := net.Listen("unix", socket)
	assert.NilError(t, err)
	defer listener.Close()

	target := filepath.Join(t.TempDir(), "out.zip")
	file, err := os.Create(target)
	assert.NilError(t, err)

	var skipped []string
	assert.NilError(t, ZipFolder(file, folder, target, func(path string) { skipped = append(skipped, path) }))
	assert.NilError(t, file.Close())

	assert.DeepEqual(t, skipped, []string{fifo, socket})
	assert.DeepEqual(t, zipNames(t, target), []string{"out/", "out/a.txt"})
}

func TestZipFolderFailsOnAnUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads any file")
	}

	folder := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(folder, "secret"), []byte("x"), 0o000))

	assert.ErrorContains(t, ZipFolder(io.Discard, folder, "", noSkips(t)), "permission denied")
}
