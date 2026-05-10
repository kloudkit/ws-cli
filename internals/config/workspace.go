package config

import (
	"fmt"
	"os"
)

func IsWorkspace() bool {
	info, err := os.Stat(DefaultManifestPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func RequireWorkspace() error {
	if !IsWorkspace() {
		return fmt.Errorf("this command requires a running Kloud Workspace")
	}
	return nil
}
