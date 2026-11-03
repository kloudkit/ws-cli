package seed

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"
)

func withNewline(data []byte) []byte {
	if len(data) == 0 || data[len(data)-1] == '\n' {
		return data
	}

	return slices.Concat(data, []byte("\n"))
}

func ensureBlock(existing, body []byte, comment string) ([]byte, error) {
	comment = cmp.Or(comment, "#")
	beginMarker, endMarker := comment+" >>> ws-seed >>>", comment+" <<< ws-seed <<<"
	block := slices.Concat([]byte(beginMarker+"\n"), withNewline(body), []byte(endMarker+"\n"))
	lines := bytes.SplitAfter(existing, []byte("\n"))

	begin, end := -1, -1
	for i, line := range lines {
		switch string(bytes.TrimRight(line, "\n")) {
		case beginMarker:
			if begin >= 0 {
				return nil, errors.New("malformed managed block: duplicate begin marker")
			}
			begin = i
		case endMarker:
			if end >= 0 {
				return nil, errors.New("malformed managed block: duplicate end marker")
			}
			end = i
		}
	}

	switch {
	case begin < 0 && end < 0:
		return slices.Concat(withNewline(existing), block), nil
	case begin < 0 || end < begin:
		return nil, errors.New("malformed managed block: markers out of order")
	}

	return bytes.Join(slices.Concat(lines[:begin], [][]byte{block}, lines[end+1:]), nil), nil
}

func lineKey(line []byte) string {
	trimmed := bytes.TrimLeft(bytes.TrimRight(line, "\n"), " \t")
	if i := bytes.IndexByte(trimmed, '='); i >= 0 {
		trimmed = trimmed[:i+1]
	}

	return string(trimmed)
}

func ensureLine(existing, body []byte) ([]byte, error) {
	managed := bytes.TrimRight(body, "\n")
	switch {
	case len(managed) == 0:
		return nil, errors.New("op: lineinfile requires content")
	case bytes.ContainsRune(managed, '\n'):
		return nil, errors.New("op: lineinfile manages a single line; use op: block for multiple")
	}

	line, key := withNewline(managed), lineKey(managed)
	lines := bytes.SplitAfter(existing, []byte("\n"))

	for i := range lines {
		if lineKey(lines[i]) == key {
			lines[i] = line
			return bytes.Join(lines, nil), nil
		}
	}

	return slices.Concat(withNewline(existing), line), nil
}

type codec struct {
	unmarshal func([]byte, any) error
	marshal   func(any) ([]byte, error)
}

func codecFor(dest string) (codec, error) {
	switch strings.ToLower(filepath.Ext(dest)) {
	case ".json":
		return codec{unmarshalJSON, marshalJSON}, nil
	case ".yaml", ".yml":
		return codec{yaml.Unmarshal, yaml.Marshal}, nil
	case ".toml":
		return codec{toml.Unmarshal, toml.Marshal}, nil
	}

	return codec{}, fmt.Errorf("cannot infer merge format from %q", dest)
}

func (c codec) decode(data []byte) (map[string]any, error) {
	var out map[string]any
	if len(bytes.TrimSpace(data)) > 0 {
		if err := c.unmarshal(data, &out); err != nil {
			return nil, err
		}
	}

	if out == nil {
		out = map[string]any{}
	}

	return out, nil
}

func unmarshalJSON(data []byte, v any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	return decoder.Decode(v)
}

func marshalJSON(v any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(v)

	return buffer.Bytes(), err
}

func mergeContent(existing, fragment []byte, dest string) ([]byte, error) {
	c, err := codecFor(dest)
	if err != nil {
		return nil, err
	}

	dst, err := c.decode(existing)
	if err != nil {
		return nil, fmt.Errorf("failed to decode existing %q", dest)
	}

	src, err := c.decode(fragment)
	if err != nil {
		return nil, fmt.Errorf("failed to decode fragment for %q", dest)
	}

	if err := deepMerge(dst, src); err != nil {
		return nil, err
	}

	return c.marshal(dst)
}

func deepMerge(dst, src map[string]any) error {
	for key, srcVal := range src {
		dstVal, exists := dst[key]
		srcMap, srcIsMap := srcVal.(map[string]any)
		dstMap, dstIsMap := dstVal.(map[string]any)

		switch {
		case exists && srcIsMap != dstIsMap:
			return fmt.Errorf("merge conflict at key %q: type mismatch", key)
		case exists && srcIsMap:
			if err := deepMerge(dstMap, srcMap); err != nil {
				return err
			}
		default:
			dst[key] = srcVal
		}
	}

	return nil
}
