package editor

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/kloudkit/ws-cli/internals/net"
)

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
	NewWindow bool   `json:"newWindow"`
	Preview   bool   `json:"preview"`
	Modal     bool   `json:"modal,omitempty"`
	Selection *Range `json:"selection,omitempty"`
}

type TerminalRequest struct {
	Cwd     string `json:"cwd"`
	Name    string `json:"name,omitempty"`
	Command string `json:"command,omitempty"`
}

type DownloadResult struct {
	Downloaded bool `json:"downloaded"`
}

type NotifyRequest struct {
	Message  string `json:"message"`
	Detail   string `json:"detail,omitempty"`
	Actions  any    `json:"actions,omitempty"`
	Modal    bool   `json:"modal,omitempty"`
	Timeout  *int   `json:"timeout,omitempty"`
	Severity string `json:"severity,omitempty"`
}

func NormalizePath(path string) string {
	if strings.HasPrefix(path, `\`) {
		return strings.ReplaceAll(path, `\`, "/")
	}

	return path
}

func DecodeTabs(body []byte) ([]Tab, error) {
	var tabs []Tab
	if err := json.Unmarshal(body, &tabs); err != nil {
		return nil, err
	}

	for i := range tabs {
		tabs[i].Path = NormalizePath(tabs[i].Path)
	}

	return tabs, nil
}

func DecodeSelection(body []byte) (*Selection, error) {
	var selection Selection
	if err := json.Unmarshal(body, &selection); err != nil {
		return nil, err
	}

	selection.Path = NormalizePath(selection.Path)

	return &selection, nil
}

func FetchDiagnostics(uri string) ([]byte, error) {
	envelope := map[string]any{"type": "diagnostics"}
	if uri != "" {
		envelope["uri"] = uri
	}

	return fetch(envelope)
}

func FetchEditors() ([]byte, error) {
	return fetch(map[string]any{"type": "editorList"})
}

func FetchSelection() ([]byte, error) {
	return fetch(map[string]any{"type": "editorSelection"})
}

func Open(req OpenRequest) error {
	_, err := net.SendEnvelope(req.Path, struct {
		Type string `json:"type"`
		OpenRequest
	}{"editorOpen", req})

	return err
}

func Reveal(path string) error {
	_, err := fetchWithin(path, map[string]any{"type": "editorReveal", "path": path}, 3*time.Second)
	return err
}

func OpenTerminal(req TerminalRequest) error {
	_, err := fetchWithin(req.Cwd, struct {
		Type string `json:"type"`
		TerminalRequest
	}{"editorTerminal", req}, 3*time.Second)

	return err
}

var DownloadWait = 310 * time.Second

func Download(path string) ([]byte, error) {
	return fetchWithin(path, map[string]any{"type": "editorDownload", "path": path}, DownloadWait)
}

func Notify(req NotifyRequest) ([]byte, error) {
	wait := time.Duration(0)
	if req.Timeout == nil {
		wait = 35 * time.Second
	} else if *req.Timeout > 0 {
		wait = time.Duration(*req.Timeout+5) * time.Second
	}

	return fetchWithin("", struct {
		Type string `json:"type"`
		NotifyRequest
	}{"notify", req}, wait)
}

func fetch(envelope any) ([]byte, error) {
	return fetchWithin("", envelope, 3*time.Second)
}

func fetchWithin(filePath string, envelope any, wait time.Duration) ([]byte, error) {
	body, err := net.SendEnvelopeWithin(filePath, envelope, wait)
	if err != nil {
		return nil, err
	}

	if string(bytes.TrimSpace(body)) == "null" {
		return nil, nil
	}

	return body, nil
}
