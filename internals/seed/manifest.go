package seed

import (
	"cmp"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

const manifestName = ".seed.yaml"

type Op string

const (
	OpCopy       Op = "copy"
	OpMerge      Op = "merge"
	OpAppend     Op = "append"
	OpPrepend    Op = "prepend"
	OpBlock      Op = "block"
	OpLineInfile Op = "lineinfile"
)

var knownOps = []Op{OpCopy, OpMerge, OpAppend, OpPrepend, OpBlock, OpLineInfile}

type SeedOp struct {
	Mode     string  `yaml:"mode"`
	Content  *string `yaml:"content"`
	Secret   bool    `yaml:"secret"`
	Op       Op      `yaml:"op"`
	Template bool    `yaml:"template"`
	Force    bool    `yaml:"force"`
	Comment  string  `yaml:"comment"`
}

type Manifest struct {
	Version string            `yaml:"version"`
	Secrets map[string]string `yaml:"secrets"`
	Seeds   map[string]SeedOp `yaml:"seeds"`
}

func manifestPath(source string) string {
	return filepath.Join(source, manifestName)
}

func parseManifest(data []byte) (*Manifest, error) {
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if manifest.Version != "v1" {
		return nil, fmt.Errorf("unsupported manifest version %q (expected \"v1\")", manifest.Version)
	}

	for name, value := range manifest.Secrets {
		if !strings.HasPrefix(value, "file:") && !strings.Contains(value, "$") {
			return nil, fmt.Errorf("secret %q: expected ciphertext or file: ref", name)
		}
	}

	for dest, op := range manifest.Seeds {
		if err := op.validate(); err != nil {
			return nil, fmt.Errorf("seed %q: %w", dest, err)
		}

		op.Op = cmp.Or(op.Op, OpCopy)
		manifest.Seeds[dest] = op
	}

	return &manifest, nil
}

func (o SeedOp) validate() error {
	switch {
	case !o.Secret && o.Mode == "" && cmp.Or(o.Op, OpCopy) == OpCopy && !o.Template && o.Content == nil:
		return errors.New("a copy-only entry is not allowed (use the mirror tier)")
	case !slices.Contains(knownOps, cmp.Or(o.Op, OpCopy)):
		return fmt.Errorf("unknown op %q", o.Op)
	case o.Comment != "" && o.Op != OpBlock:
		return errors.New("comment is only valid with op: block")
	}

	return nil
}
