package io

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func ZipFolder(dst io.Writer, folder, skip string) error {
	archive := zip.NewWriter(dst)
	parent := filepath.Dir(folder)

	err := filepath.WalkDir(folder, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || path == skip {
			return err
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		name, err := filepath.Rel(parent, path)
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(name)

		switch mode := info.Mode(); {
		case mode.IsDir():
			header.Name += "/"
			_, err = archive.CreateHeader(header)
			return err
		case mode&fs.ModeSymlink != 0:
			return addLink(archive, header, path)
		case mode.IsRegular():
			header.Method = zip.Deflate
			return addFile(archive, header, path)
		}

		return fmt.Errorf("cannot zip %s: not a regular file, folder or link", path)
	})
	if err != nil {
		return err
	}

	return archive.Close()
}

func addLink(archive *zip.Writer, header *zip.FileHeader, path string) error {
	target, err := os.Readlink(path)
	if err != nil {
		return err
	}

	entry, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.WriteString(entry, target)
	return err
}

func addFile(archive *zip.Writer, header *zip.FileHeader, path string) error {
	source, err := os.Open(path)
	if err != nil {
		return err
	}
	defer source.Close()

	entry, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(entry, source)
	return err
}
