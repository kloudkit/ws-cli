package io

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"gotest.tools/v3/assert"
)

func TestZipFolderSkipsItsOwnOutput(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "out")
	assert.NilError(t, os.Mkdir(folder, 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(folder, "a.txt"), []byte("alpha"), 0o600))

	target := filepath.Join(folder, "self.zip")
	file, err := os.Create(target)
	assert.NilError(t, err)

	assert.NilError(t, ZipFolder(file, folder, target))
	assert.NilError(t, file.Close())

	archive, err := zip.OpenReader(target)
	assert.NilError(t, err)
	defer archive.Close()

	var names []string
	for _, entry := range archive.File {
		names = append(names, entry.Name)
	}
	assert.DeepEqual(t, names, []string{"out/", "out/a.txt"})
}

func TestZipFolderRefusesSpecialFiles(t *testing.T) {
	folder := t.TempDir()
	fifo := filepath.Join(folder, "pipe")
	assert.NilError(t, syscall.Mkfifo(fifo, 0o600))

	err := ZipFolder(io.Discard, folder, "")

	assert.ErrorContains(t, err, "cannot zip "+fifo+": not a regular file, folder or link")
}
