package seed

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kloudkit/ws-cli/internals/secrets"
	"gopkg.in/yaml.v3"
)

const rotateProbeName = ".ws-rotate-probe"

type RotateOptions struct {
	Source       string
	MasterKey    string
	NewMasterKey string
	Out          io.Writer
	Styled       bool
}

type rotateTarget struct {
	describe  string
	cipher    string
	plain     []byte
	writePath string
	writeBack func(string) error
}

func Rotate(opts RotateOptions) error {
	if opts.NewMasterKey == "" {
		return fmt.Errorf("a new master key is required (use --new-master)")
	}

	manifestPath := ManifestPath(opts.Source)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest %q: %w", manifestPath, err)
	}

	manifest, err := ParseManifest(raw)
	if err != nil {
		return err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	oldKey, err := secrets.ResolveMasterKey(opts.MasterKey)
	if err != nil {
		return err
	}
	defer clear(oldKey)

	newKey, err := secrets.ResolveMasterKey(opts.NewMasterKey)
	if err != nil {
		return err
	}
	defer clear(newKey)

	targets, err := collectTargets(opts.Source, manifest, &doc)
	if err != nil {
		return err
	}

	defer func() {
		for i := range targets {
			clear(targets[i].plain)
		}
	}()

	for i := range targets {
		plain, err := secrets.Decrypt(secrets.NormalizeEncrypted(targets[i].cipher), oldKey)
		if err != nil {
			return fmt.Errorf("%s: decrypt failed (wrong current key?)", targets[i].describe)
		}

		targets[i].plain = plain
	}

	writeManifest := slices.ContainsFunc(targets, func(t rotateTarget) bool { return t.writePath == "" })

	if err := preflightWritable(targets, manifestPath, writeManifest); err != nil {
		return err
	}

	for i := range targets {
		reencrypted, err := secrets.Encrypt(targets[i].plain, newKey)
		if err != nil {
			return fmt.Errorf("%s: re-encrypt failed", targets[i].describe)
		}

		if err := targets[i].writeBack(reencrypted); err != nil {
			return fmt.Errorf("%s: %w", targets[i].describe, err)
		}
	}

	if writeManifest {
		if err := writeManifestFile(manifestPath, &doc); err != nil {
			return err
		}
	}

	rep := reporter{out: opts.Out, styled: opts.Styled}
	for i := range targets {
		rep.rotated(targets[i].describe)
	}

	return nil
}

func collectTargets(source string, manifest *Manifest, doc *yaml.Node) ([]rotateTarget, error) {
	root := documentRoot(doc)
	vars := resolveVars()

	var targets []rotateTarget

	for _, name := range slices.Sorted(maps.Keys(manifest.Secrets)) {
		target, err := valueTarget(
			fmt.Sprintf("secret %q", name),
			manifest.Secrets[name],
			nodeSetter(root, "secrets", name),
		)
		if err != nil {
			return nil, err
		}

		targets = append(targets, target)
	}

	for _, rawDest := range slices.Sorted(maps.Keys(manifest.Seeds)) {
		op := manifest.Seeds[rawDest]
		if !op.Secret {
			continue
		}

		if op.Content != nil {
			target, err := valueTarget(
				fmt.Sprintf("seed %q", rawDest),
				*op.Content,
				nodeSetter(root, "seeds", rawDest, "content"),
			)
			if err != nil {
				return nil, err
			}

			targets = append(targets, target)
			continue
		}

		target, err := fileTarget(fmt.Sprintf("seed %q", rawDest), rhymingSource(source, vars.expand(rawDest)))
		if err != nil {
			return nil, err
		}

		targets = append(targets, target)
	}

	return targets, nil
}

func valueTarget(describe, value string, inline func(string) error) (rotateTarget, error) {
	if path, ok := fileRef(value); ok {
		return fileTarget(describe, path)
	}

	return rotateTarget{
		describe:  describe,
		cipher:    secrets.NormalizeEncrypted(value),
		writeBack: inline,
	}, nil
}

func fileTarget(describe, path string) (rotateTarget, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return rotateTarget{}, fmt.Errorf("%s: failed to read %q: %w", describe, path, err)
	}

	normalized := secrets.NormalizeEncrypted(string(data))
	if ref, ok := fileRef(normalized); ok {
		return fileTarget(describe, ref)
	}

	return rotateTarget{
		describe:  describe,
		cipher:    normalized,
		writePath: path,
		writeBack: fileWriter(path),
	}, nil
}

func fileRef(value string) (string, bool) {
	path, ok := strings.CutPrefix(value, "file:")

	return path, ok && path != ""
}

func fileWriter(path string) func(string) error {
	return func(reencrypted string) error {
		return atomicReplace(path, []byte(reencrypted))
	}
}

func nodeSetter(root *yaml.Node, path ...string) func(string) error {
	return func(reencrypted string) error {
		node := root
		for _, key := range path {
			node = mappingValue(node, key)
		}

		if node == nil {
			return fmt.Errorf("manifest node %s not found", strings.Join(path, "."))
		}

		node.Value = reencrypted
		node.Tag = "!!str"

		return nil
	}
}

func preflightWritable(targets []rotateTarget, manifestPath string, writeManifest bool) error {
	paths := []string{}
	for i := range targets {
		if targets[i].writePath != "" {
			paths = append(paths, targets[i].writePath)
		}
	}

	if writeManifest {
		paths = append(paths, manifestPath)
	}

	dirs := map[string]bool{}
	for _, p := range paths {
		if info, err := os.Lstat(p); err == nil && info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("refusing to write through symlink %q", p)
		}
		dirs[filepath.Dir(p)] = true
	}

	for dir := range dirs {
		probe := filepath.Join(dir, rotateProbeName)
		if err := os.WriteFile(probe, []byte{}, 0o600); err != nil {
			return fmt.Errorf("cannot write to %q: %w", dir, err)
		}
		os.Remove(probe)
	}

	return nil
}

func documentRoot(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}

	return doc
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}

	return nil
}

func writeManifestFile(path string, doc *yaml.Node) error {
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)

	if err := encoder.Encode(doc); err != nil {
		return fmt.Errorf("failed to encode manifest: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to encode manifest: %w", err)
	}

	return atomicReplace(path, buffer.Bytes())
}

func atomicReplace(path string, data []byte) error {
	perm := fs.FileMode(0o600)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}

	return writeAtomic(filepath.Dir(path), path, data, perm)
}
