package config

import (
	"bufio"
	"bytes"
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
	scanner := bufio.NewScanner(bytes.NewReader(out))

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "@")

		if len(parts) == 2 {
			extensions = append(extensions, Extension{
				Name:    parts[0],
				Version: parts[1],
			})
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
