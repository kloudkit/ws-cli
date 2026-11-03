package config

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var extensionsTimeout = 10 * time.Second

type Extension struct {
	Name    string
	Version string
}

func GetExtensions() ([]Extension, error) {
	ctx, cancel := context.WithTimeout(context.Background(), extensionsTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "code", "--list-extensions", "--show-versions")
	cmd.WaitDelay = time.Second

	out, err := cmd.Output()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
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
