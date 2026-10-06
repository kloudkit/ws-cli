package seed

import (
	"bytes"
	"fmt"
	"strings"
)

func lineKey(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	if i := strings.IndexByte(trimmed, '='); i >= 0 {
		return trimmed[:i+1]
	}

	return trimmed
}

func ensureLine(existing, body []byte) ([]byte, error) {
	managed := bytes.TrimRight(body, "\n")
	if len(managed) == 0 {
		return nil, fmt.Errorf("op: lineinfile requires content")
	}

	if bytes.Contains(managed, []byte("\n")) {
		return nil, fmt.Errorf("op: lineinfile manages a single line; use op: block for multiple")
	}

	line := append(append([]byte(nil), managed...), '\n')
	key := lineKey(string(managed))
	lines := bytes.SplitAfter(existing, []byte("\n"))

	for i, existingLine := range lines {
		if lineKey(strings.TrimRight(string(existingLine), "\n")) != key {
			continue
		}

		lines[i] = line

		return bytes.Join(lines, nil), nil
	}

	return appendBlock(existing, line), nil
}
