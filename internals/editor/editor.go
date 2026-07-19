package editor

import (
	"bytes"

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
	Window    string `json:"window"`
	Preview   bool   `json:"preview,omitempty"`
	Selection *Range `json:"selection,omitempty"`
}

type NotifyRequest struct {
	Message  string `json:"message"`
	Detail   string `json:"detail,omitempty"`
	Actions  any    `json:"actions,omitempty"`
	Modal    bool   `json:"modal,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`
	Severity string `json:"severity,omitempty"`
}

func FetchDiagnostics(uri string) ([]byte, error) {
	envelope := map[string]any{"type": "diagnostics"}
	if uri != "" {
		envelope["uri"] = uri
	}

	return fetch("", envelope)
}

func FetchEditors() ([]byte, error) {
	return fetch("", map[string]any{"type": "editorList"})
}

func FetchSelection() ([]byte, error) {
	return fetch("", map[string]any{"type": "editorSelection"})
}

func Open(req OpenRequest) error {
	envelope := map[string]any{
		"type":      "editorOpen",
		"path":      req.Path,
		"preview":   req.Preview,
		"newWindow": req.Window == "new",
	}

	if req.Selection != nil {
		envelope["selection"] = req.Selection
	}

	_, err := net.SendEnvelope(req.Path, envelope)

	return err
}

func Notify(req NotifyRequest) ([]byte, error) {
	envelope := map[string]any{"type": "notify", "message": req.Message}

	if req.Detail != "" {
		envelope["detail"] = req.Detail
	}
	if req.Actions != nil {
		envelope["actions"] = req.Actions
	}
	if req.Modal {
		envelope["modal"] = true
	}
	if req.Timeout > 0 {
		envelope["timeout"] = req.Timeout
	}
	if req.Severity != "" {
		envelope["severity"] = req.Severity
	}

	return fetch("", envelope)
}

func fetch(filePath string, envelope any) ([]byte, error) {
	body, err := net.SendEnvelope(filePath, envelope)
	if err != nil {
		return nil, err
	}

	if string(bytes.TrimSpace(body)) == "null" {
		return nil, nil
	}

	return body, nil
}
