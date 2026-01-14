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
	file, ok := reader.(*os.File)
	if !ok {
		return readPasswordFromStream(reader)
	}

	stat, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat stdin: %w", err)
	}

	isTerminal := (stat.Mode() & os.ModeCharDevice) != 0
	if !isTerminal {
		return readPasswordFromStream(file)
	}

	fmt.Fprint(os.Stderr, "Enter password: ")
	passwordBytes, err := term.ReadPassword(int(file.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	return string(passwordBytes), nil
}

func readPasswordFromStream(reader io.Reader) (string, error) {
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
