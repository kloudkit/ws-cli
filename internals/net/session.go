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
	"cannot reach the workspace editor; open an editor session first",
)

var errEditorTooOld = errors.New(
	"the running editor does not support this command yet; it needs a newer workspace image",
)

func SessionSocketPath() string {
	base := filepath.Join(env.Home(), ".local", "share")

	if xdg := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(xdg) {
		base = xdg
	}

	return filepath.Join(base, "ws-server", "session.sock")
}

const answerWithin = 3 * time.Second

func unixClient(socket string, wait time.Duration) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (nativeNet.Conn, error) {
				return (&nativeNet.Dialer{Timeout: answerWithin}).DialContext(ctx, "unix", socket)
			},
		},
		Timeout: wait,
	}
}

func requestError(err error, wait time.Duration) error {
	var opErr *nativeNet.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return errNoEditor
	}

	var netErr nativeNet.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("the workspace editor did not answer within %s", wait)
	}

	return fmt.Errorf("error talking to the workspace editor: %w", err)
}

func resolveWindowSocket(filePath string) (string, error) {
	if hook := os.Getenv("VSCODE_IPC_HOOK_CLI"); hook != "" {
		return hook, nil
	}

	target := "http://unix/session?filePath=" + url.QueryEscape(filePath)

	resp, err := unixClient(SessionSocketPath(), answerWithin).Get(target)
	if err != nil {
		return "", requestError(err, answerWithin)
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
	return SendEnvelopeWithin(filePath, envelope, answerWithin)
}

func SendEnvelopeWithin(filePath string, envelope any, wait time.Duration) ([]byte, error) {
	socket, err := resolveWindowSocket(filePath)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("error encoding editor request: %w", err)
	}

	resp, err := unixClient(socket, wait).Post("http://unix/", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, requestError(err, wait)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading editor response: %w", err)
	}

	if resp.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(data))
		if text := ""; json.Unmarshal(data, &text) == nil {
			message = text
		}

		if message == "" {
			message = resp.Status
		}

		if resp.StatusCode == http.StatusNotFound && strings.HasPrefix(message, "Unknown message type") {
			return nil, errEditorTooOld
		}

		return nil, fmt.Errorf("workspace editor returned %d: %s", resp.StatusCode, message)
	}

	return data, nil
}

var errNoServer = errors.New(
	"cannot reach the workspace server's session socket; is the editor running?",
)

func sessionError(resp *http.Response) error {
	data, _ := io.ReadAll(resp.Body)

	var payload struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(data, &payload)

	if resp.StatusCode == http.StatusNotFound && payload.Error == "not found" {
		return errEditorTooOld
	}

	if payload.Error == "" {
		payload.Error = strings.TrimSpace(string(data))
	}

	if payload.Error == "" {
		payload.Error = resp.Status
	}

	return errors.New(payload.Error)
}

func serverRequestError(err error) error {
	var opErr *nativeNet.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return errNoServer
	}

	return requestError(err, answerWithin)
}

func sessionRequest(client *http.Client, method, path string, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error encoding workspace server request: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, "http://unix"+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, serverRequestError(err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, sessionError(resp)
	}

	return resp, nil
}

func SessionCall(method, path string, body any) ([]byte, error) {
	resp, err := sessionRequest(unixClient(SessionSocketPath(), answerWithin), method, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func SessionStream(path string, body any) (io.ReadCloser, error) {
	resp, err := sessionRequest(unixClient(SessionSocketPath(), 0), http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}
