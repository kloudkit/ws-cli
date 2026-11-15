package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kloudkit/ws-cli/internals/proc"
)

var extensionsTimeout = 10 * time.Second

type Extension struct {
	Name    string
	Version string
}

func GetExtensions() ([]Extension, error) {
	out, err := proc.Output(extensionsTimeout, "code", "--list-extensions", "--show-versions")
	if errors.Is(err, proc.ErrTimeout) {
		return nil, fmt.Errorf("`code --list-extensions` did not answer within %s", extensionsTimeout)
	}

	if err != nil {
		return nil, fmt.Errorf("`code --list-extensions` failed: %w", err)
	}

	var extensions []Extension
	for line := range strings.Lines(string(out)) {
		name, version, ok := strings.Cut(strings.TrimSpace(line), "@")
		if ok && !strings.Contains(version, "@") {
			extensions = append(extensions, Extension{Name: name, Version: version})
		}
	}

	return extensions, nil
}

func GetExtensionCount() (int, error) {
	extensions, err := GetExtensions()
	if err != nil {
		return 0, err
	}

	return len(extensions), nil
}
