package config

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"
)

type Extension struct {
	Name    string
	Version string
}

func GetExtensions() ([]Extension, error) {
	out, err := exec.Command("code", "--list-extensions", "--show-versions").Output()
	if err != nil {
		return nil, err
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
