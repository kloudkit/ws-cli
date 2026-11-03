package seed

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"go.yaml.in/yaml/v3"
)

const rotateProbeName = ".ws-rotate-probe"

type RotateOptions struct {
	Source       string
	MasterKey    string
	NewMasterKey string
	Out          io.Writer
}

type rotateTarget struct {
	describe string
	cipher   string
	plain    []byte
	path     string
	node     []string
}

func Rotate(opts RotateOptions) error {
	if opts.NewMasterKey == "" {
		return errors.New("a new master key is required (use --new-master)")
	}

	file := manifestPath(opts.Source)
	raw, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read manifest %q: %w", file, err)
	}

	manifest, err := parseManifest(raw)
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

	targets, err := collectTargets(opts.Source, manifest)
	defer func() {
		for i := range targets {
			clear(targets[i].plain)
		}
	}()

	if err != nil {
		return err
	}

	for i := range targets {
		if targets[i].plain, err = secrets.Decrypt(targets[i].cipher, oldKey); err != nil {
			return fmt.Errorf("%s: decrypt failed (wrong current key?)", targets[i].describe)
		}
	}

	if err := preflightWritable(targets, file); err != nil {
		return err
	}

	root, writeManifest := &doc, false
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		root = doc.Content[0]
	}

	for _, target := range targets {
		reencrypted, err := secrets.Encrypt(target.plain, newKey)
		if err != nil {
			return fmt.Errorf("%s: re-encrypt failed", target.describe)
		}

		if target.path != "" {
			err = atomicReplace(target.path, []byte(reencrypted))
		} else {
			err = setNode(root, reencrypted, target.node)
			writeManifest = true
		}

		if err != nil {
			return fmt.Errorf("%s: %w", target.describe, err)
		}
	}

	if writeManifest {
		var buffer bytes.Buffer
		encoder := yaml.NewEncoder(&buffer)
		encoder.SetIndent(2)

		if err := errors.Join(encoder.Encode(&doc), encoder.Close()); err != nil {
			return fmt.Errorf("failed to encode manifest: %w", err)
		}

		if err := atomicReplace(file, buffer.Bytes()); err != nil {
			return err
		}
	}

	for _, target := range targets {
		styles.PrintSuccess(opts.Out, "Rotated "+target.describe)
	}

	return nil
}

func collectTargets(source string, manifest *Manifest) ([]rotateTarget, error) {
	vars := resolveVars()

	var targets []rotateTarget
	for _, name := range slices.Sorted(maps.Keys(manifest.Secrets)) {
		targets = append(targets, rotateTarget{
			describe: fmt.Sprintf("secret %q", name),
			cipher:   manifest.Secrets[name],
			node:     []string{"secrets", name},
		})
	}

	for _, rawDest := range slices.Sorted(maps.Keys(manifest.Seeds)) {
		op := manifest.Seeds[rawDest]
		target := rotateTarget{describe: fmt.Sprintf("seed %q", rawDest)}

		switch {
		case !op.Secret:
			continue
		case op.Content != nil:
			target.cipher, target.node = *op.Content, []string{"seeds", rawDest, "content"}
		default:
			target.path = rhymingSource(source, vars.expand(rawDest))
		}

		targets = append(targets, target)
	}

	for i := range targets {
		if err := targets[i].load(); err != nil {
			return targets, err
		}
	}

	return targets, nil
}

func (t *rotateTarget) load() error {
	if t.path != "" {
		if err := t.read(t.path); err != nil {
			return err
		}
	}

	if ref, ok := strings.CutPrefix(t.cipher, "file:"); ok && ref != "" {
		return t.read(ref)
	}

	t.cipher = secrets.NormalizeEncrypted(t.cipher)

	return nil
}

func (t *rotateTarget) read(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: failed to read %q: %w", t.describe, path, err)
	}

	t.path, t.cipher = path, secrets.NormalizeEncrypted(string(data))

	return nil
}

func setNode(root *yaml.Node, value string, keys []string) error {
	node := root
	for _, key := range keys {
		node = mappingValue(node, key)
	}

	if node == nil {
		return fmt.Errorf("manifest node %s not found", strings.Join(keys, "."))
	}

	node.Value, node.Tag = value, "!!str"

	return nil
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

func preflightWritable(targets []rotateTarget, manifest string) error {
	dirs := map[string]bool{}
	for _, target := range targets {
		p := cmp.Or(target.path, manifest)
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

func atomicReplace(path string, data []byte) error {
	perm := fs.FileMode(0o600)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}

	return writeAtomic(filepath.Dir(path), path, data, perm)
}
