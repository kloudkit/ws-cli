package clipboard

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/kloudkit/ws-cli/internals/net"
)

func Paste(writer io.Writer) error {
	body, err := net.SendEnvelope("", map[string]any{"type": "clipboardRead"})
	if err != nil {
		return err
	}

	var text string
	if err := json.Unmarshal(body, &text); err != nil {
		return fmt.Errorf("error decoding clipboard response: %w", err)
	}

	if _, err := io.WriteString(writer, text); err != nil {
		return fmt.Errorf("error outputting clipboard data: %w", err)
	}

	return nil
}

func Copy(reader io.Reader) error {
	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("error reading clipboard input: %w", err)
	}

	_, err = net.SendEnvelope("", map[string]any{
		"type":    "clipboard",
		"content": string(content),
	})

	return err
}
