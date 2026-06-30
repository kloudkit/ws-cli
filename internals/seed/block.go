package seed

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	blockBeginBody = ">>> ws-seed >>>"
	blockEndBody   = "<<< ws-seed <<<"
	defaultComment = "#"
	blockBegin     = defaultComment + " " + blockBeginBody
	blockEnd       = defaultComment + " " + blockEndBody
)

func blockMarkers(comment string) (string, string) {
	if comment == "" {
		comment = defaultComment
	}

	return comment + " " + blockBeginBody, comment + " " + blockEndBody
}

func renderBlock(body []byte, begin, end string) []byte {
	var buffer bytes.Buffer
	buffer.WriteString(begin)
	buffer.WriteByte('\n')

	if len(body) > 0 {
		buffer.Write(body)
		if body[len(body)-1] != '\n' {
			buffer.WriteByte('\n')
		}
	}

	buffer.WriteString(end)
	buffer.WriteByte('\n')

	return buffer.Bytes()
}

func appendBlock(existing, block []byte) []byte {
	if len(existing) == 0 {
		return block
	}

	var buffer bytes.Buffer
	buffer.Write(existing)
	if existing[len(existing)-1] != '\n' {
		buffer.WriteByte('\n')
	}
	buffer.Write(block)

	return buffer.Bytes()
}

func ensureBlock(existing, body []byte, comment string) ([]byte, error) {
	beginMarker, endMarker := blockMarkers(comment)
	block := renderBlock(body, beginMarker, endMarker)
	lines := bytes.SplitAfter(existing, []byte("\n"))

	begin, end := -1, -1
	for i, line := range lines {
		switch strings.TrimRight(string(line), "\n") {
		case beginMarker:
			if begin >= 0 {
				return nil, fmt.Errorf("malformed managed block: duplicate begin marker")
			}
			begin = i
		case endMarker:
			if end >= 0 {
				return nil, fmt.Errorf("malformed managed block: duplicate end marker")
			}
			end = i
		}
	}

	if begin < 0 && end < 0 {
		return appendBlock(existing, block), nil
	}

	if begin < 0 || end < 0 || end < begin {
		return nil, fmt.Errorf("malformed managed block: markers out of order")
	}

	var buffer bytes.Buffer
	for _, line := range lines[:begin] {
		buffer.Write(line)
	}
	buffer.Write(block)
	for _, line := range lines[end+1:] {
		buffer.Write(line)
	}

	return buffer.Bytes(), nil
}
