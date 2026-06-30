package seed

import (
	"io/fs"
	"os"
	"path/filepath"
)

func walkMirror(source string) (map[string]string, error) {
	mirror := map[string]string{}

	info, err := os.Stat(source)
	if err != nil || !info.IsDir() {
		return mirror, nil
	}

	err = filepath.WalkDir(source, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(source, p)
		if err != nil {
			return err
		}

		if rel == ManifestName {
			return nil
		}

		mirror[string(os.PathSeparator)+rel] = p

		return nil
	})

	if err != nil {
		return nil, err
	}

	return mirror, nil
}
