package config

import (
	"fmt"
	"os"
)

func Bootstrap() error {
	if info, err := os.Stat(DefaultManifestPath); err != nil || info.IsDir() {
		return fmt.Errorf("this command requires a running Kloud Workspace")
	}

	_, err := LoadEnvReference()

	return err
}
