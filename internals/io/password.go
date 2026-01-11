package io

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func ReadPasswordInput() (string, error) {
	return ReadPasswordFromReader(os.Stdin)
}

func ReadPasswordFromReader(reader io.Reader) (string, error) {
	if file, ok := reader.(*os.File); ok {
		stat, err := file.Stat()
		if err != nil {
			return "", fmt.Errorf("failed to stat stdin: %w", err)
		}

		if (stat.Mode() & os.ModeCharDevice) == 0 {
			data, err := io.ReadAll(file)
			if err != nil {
				return "", fmt.Errorf("failed to read from stdin: %w", err)
			}
			password := string(data)
			if strings.TrimSpace(password) == "" {
				return "", fmt.Errorf("empty password provided")
			}
			return password, nil
		}

		fmt.Fprint(os.Stderr, "Enter password: ")
		passwordBytes, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}

		return string(passwordBytes), nil
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read from stdin: %w", err)
	}
	password := string(data)
	if strings.TrimSpace(password) == "" {
		return "", fmt.Errorf("empty password provided")
	}
	return password, nil
}
