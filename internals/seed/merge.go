package seed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

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

	if err := encoder.Encode(v); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func mergeContent(existing, fragment []byte, dest string) ([]byte, error) {
	c, err := codecFor(dest)
	if err != nil {
		return nil, err
	}

	dst := map[string]any{}
	if len(bytes.TrimSpace(existing)) > 0 {
		if err := c.unmarshal(existing, &dst); err != nil {
			return nil, fmt.Errorf("failed to decode existing %q", dest)
		}
	}

	src := map[string]any{}
	if len(bytes.TrimSpace(fragment)) > 0 {
		if err := c.unmarshal(fragment, &src); err != nil {
			return nil, fmt.Errorf("failed to decode fragment for %q", dest)
		}
	}

	if err := deepMerge(dst, src); err != nil {
		return nil, err
	}

	return c.marshal(dst)
}

func deepMerge(dst, src map[string]any) error {
	for key, srcVal := range src {
		dstVal, exists := dst[key]
		if !exists {
			dst[key] = srcVal
			continue
		}

		srcMap, srcIsMap := srcVal.(map[string]any)
		dstMap, dstIsMap := dstVal.(map[string]any)

		if srcIsMap != dstIsMap {
			return fmt.Errorf("merge conflict at key %q: type mismatch", key)
		}

		if srcIsMap {
			if err := deepMerge(dstMap, srcMap); err != nil {
				return err
			}
			continue
		}

		dst[key] = srcVal
	}

	return nil
}
