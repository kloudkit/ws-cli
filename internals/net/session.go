package net

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	nativeNet "net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kloudkit/ws-cli/internals/env"
)

var errNoEditor = errors.New(
	"cannot reach the workspace editor — is an editor session open?",
)

func SessionSocketPath() string {
	base := filepath.Join(env.Home(), ".local", "share")

	if xdg := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(xdg) {
		base = xdg
	}

	return filepath.Join(base, "ws-server", "session.sock")
}

func unixClient(socket string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (nativeNet.Conn, error) {
				return (&nativeNet.Dialer{}).DialContext(ctx, "unix", socket)
			},
		},
		Timeout: 3 * time.Second,
	}
}

func resolveWindowSocket(filePath string) (string, error) {
	if hook := os.Getenv("VSCODE_IPC_HOOK_CLI"); hook != "" {
		return hook, nil
	}

	target := "http://unix/session?filePath=" + url.QueryEscape(filePath)

	resp, err := unixClient(SessionSocketPath()).Get(target)
	if err != nil {
		return "", errNoEditor
	}
	defer resp.Body.Close()

	var payload struct {
		SocketPath string `json:"socketPath"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil || payload.SocketPath == "" {
		return "", errNoEditor
	}

	return payload.SocketPath, nil
}

func SendEnvelope(filePath string, envelope any) ([]byte, error) {
	socket, err := resolveWindowSocket(filePath)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("error encoding editor request: %w", err)
	}

	resp, err := unixClient(socket).Post("http://unix/", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, errNoEditor
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading editor response: %w", err)
	}

	if resp.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = resp.Status
		}

		return nil, fmt.Errorf("workspace editor returned %d: %s", resp.StatusCode, message)
	}

	return data, nil
}
