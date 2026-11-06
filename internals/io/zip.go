package io

import (
	"archive/zip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func ZipFolder(dst io.Writer, folder, skip string, skipped func(path string)) error {
	root, err := filepath.EvalSymlinks(folder)
	if err != nil {
		return err
	}

	if resolved, err := filepath.EvalSymlinks(skip); err == nil {
		skip = resolved
	}

	archive := zip.NewWriter(dst)

	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || path == skip {
			return err
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name = filepath.Join(filepath.Base(folder), name)

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

		skipped(path)

		return nil
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
