package editor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/kloudkit/ws-cli/internals/net"
)

const base = "http://localhost"

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Severity struct {
	Code  int    `json:"code"`
	Label string `json:"label"`
}

type Diagnostic struct {
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Source   string   `json:"source"`
	Code     string   `json:"code"`
	Range    Range    `json:"range"`
}

type DiagnosticFile struct {
	URI   string       `json:"uri"`
	Items []Diagnostic `json:"items"`
}

type Tab struct {
	Path       string  `json:"path"`
	LanguageID *string `json:"languageId"`
	Active     bool    `json:"active"`
	Dirty      bool    `json:"dirty"`
}

type Selection struct {
	Path  string `json:"path"`
	Text  string `json:"text"`
	Range Range  `json:"range"`
}

type OpenRequest struct {
	Path      string `json:"path"`
	Window    string `json:"window"`
	Preview   bool   `json:"preview,omitempty"`
	Selection *Range `json:"selection,omitempty"`
}

func FetchDiagnostics(uri string) ([]byte, error) {
	query := url.Values{}
	if uri != "" {
		query.Set("uri", uri)
	}

	return read("/diagnostics", query)
}

func FetchEditors() ([]byte, error) {
	return read("/editors", nil)
}

func FetchSelection() ([]byte, error) {
	return read("/selection", nil)
}

func Open(req OpenRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error encoding open request: %w", err)
	}

	resp, err := request(http.MethodPost, base+"/open", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return checkStatus(resp)
}

func read(path string, query url.Values) ([]byte, error) {
	target := base + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	resp, err := request(http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if err := checkStatus(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading editor response: %w", err)
	}

	return body, nil
}

func request(method, target string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, target, body)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := net.GetIPCClient().Do(req)
	if err != nil {
		return nil, errors.New("cannot reach the workspace editor — is an editor session open?")
	}

	return resp, nil
}

func checkStatus(resp *http.Response) error {
	if resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = resp.Status
	}

	return fmt.Errorf("workspace editor returned %d: %s", resp.StatusCode, message)
}
