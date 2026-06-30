package seed

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const ManifestName = ".seed.yaml"

type Manifest struct {
	Version string            `yaml:"version"`
	Secrets map[string]string `yaml:"secrets"`
	Seeds   map[string]SeedOp `yaml:"seeds"`
}

func ManifestPath(source string) string {
	return filepath.Join(source, ManifestName)
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest %q: %w", path, err)
	}

	return ParseManifest(data)
}

func ParseManifest(data []byte) (*Manifest, error) {
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if manifest.Version != "v1" {
		return nil, fmt.Errorf("unsupported manifest version %q (expected \"v1\")", manifest.Version)
	}

	for name, value := range manifest.Secrets {
		if err := validateSecretValue(name, value); err != nil {
			return nil, err
		}
	}

	for dest, op := range manifest.Seeds {
		if !op.hasBehavior() {
			return nil, fmt.Errorf("seed %q: a copy-only entry is not allowed (use the mirror tier)", dest)
		}

		if op.Op == "" {
			op.Op = OpCopy
			manifest.Seeds[dest] = op
		}

		if err := validateOp(dest, op); err != nil {
			return nil, err
		}
	}

	return &manifest, nil
}

func validateSecretValue(name, value string) error {
	if value != "" && (strings.HasPrefix(value, "file:") || strings.Contains(value, "$")) {
		return nil
	}

	return fmt.Errorf("secret %q: expected ciphertext or file: ref", name)
}

func validateOp(dest string, op SeedOp) error {
	switch op.Op {
	case OpCopy, OpMerge, OpAppend, OpPrepend, OpBlock:
	default:
		return fmt.Errorf("seed %q: unknown op %q", dest, op.Op)
	}

	if op.Comment != "" && op.Op != OpBlock {
		return fmt.Errorf("seed %q: comment is only valid with op: block", dest)
	}

	return nil
}
