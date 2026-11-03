package seed

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"

	internalIO "github.com/kloudkit/ws-cli/internals/io"
	"github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
)

var errExists = errors.New("exists")

type Options struct {
	Source    string
	Force     bool
	Dests     []string
	MasterKey string
	Out       io.Writer
}

type keyResolver struct {
	flag    string
	secrets map[string]string
	key     []byte
	loaded  bool
	err     error
}

func (k *keyResolver) named(name string) ([]byte, error) {
	value, ok := k.secrets[name]
	if !ok {
		return nil, fmt.Errorf("secret %q not declared", name)
	}

	return k.decrypt(value)
}

func (k *keyResolver) decrypt(value string) ([]byte, error) {
	resolved, err := secrets.ResolveEncryptedValue(value)
	if err != nil {
		return nil, errors.New("secret source unresolved")
	}

	if !k.loaded {
		k.key, k.err = secrets.ResolveMasterKey(k.flag)
		k.loaded = true
	}

	if k.err != nil {
		return nil, errors.New("master key unavailable")
	}

	plain, err := secrets.Decrypt(secrets.NormalizeEncrypted(resolved), k.key)
	if err != nil {
		return nil, errors.New("decrypt failed")
	}

	return plain, nil
}

func Apply(opts Options) error {
	plan, err := BuildPlan(opts.Source, opts.Force)
	if err != nil {
		return err
	}

	ops := plan.Ops
	if len(opts.Dests) > 0 {
		if ops, err = plan.filterDests(opts.Dests); err != nil {
			return err
		}
	}

	if len(ops) == 0 {
		styles.PrintEmpty(opts.Out, fmt.Sprintf("No seed entries in [%s]", plan.source))
		return nil
	}

	keys := &keyResolver{flag: opts.MasterKey, secrets: plan.manifest.Secrets}
	defer func() { clear(keys.key) }()

	failures := 0
	for _, op := range ops {
		wrote, err := plan.applyOne(op, keys)

		switch {
		case errors.Is(err, errExists):
			styles.PrintWarning(opts.Out, fmt.Sprintf("Skipping [%s] (exists)", op.Dest))
		case err != nil:
			styles.PrintWarning(opts.Out, fmt.Sprintf("Skipping [%s] (%s)", op.Dest, err))
			failures++
		default:
			if wrote && consumedNotice(op.Dest, plan.vars.Home) {
				styles.PrintKeyValues(opts.Out, styles.Pair{Key: "Notice", Value: fmt.Sprintf("[%s] runs next boot; ensure +x if executable", op.Dest)})
			}

			styles.PrintSuccess(opts.Out, fmt.Sprintf("Seeded [%s]", op.Dest))
		}
	}

	switch {
	case failures == 1:
		return errors.New("1 seed entry failed to apply")
	case failures > 1:
		return fmt.Errorf("%d seed entries failed to apply", failures)
	}

	return nil
}

func (p *Plan) applyOne(op ResolvedOp, keys *keyResolver) (bool, error) {
	ancestor := nearestExistingAncestor(op.Dest)
	if !ownsPath(ancestor) {
		return false, errors.New("destination not owned")
	}

	edit := op.Op == OpBlock || op.Op == OpLineInfile
	if !edit && !internalIO.CanOverride(op.Dest, op.Force) {
		return false, errExists
	}

	var existing []byte
	if op.Op != OpCopy {
		existing, _ = os.ReadFile(op.Dest)
	}

	content, mode, err := p.materialize(op, existing, keys)
	if err != nil || edit && bytes.Equal(content, existing) {
		return false, err
	}

	return true, writeAtomic(chooseAnchor(op.Dest, p.vars, ancestor), op.Dest, content, mode)
}

func (p *Plan) materialize(op ResolvedOp, existing []byte, keys *keyResolver) ([]byte, fs.FileMode, error) {
	var raw []byte
	var err error

	if op.Content != nil {
		raw = []byte(*op.Content)
	} else if raw, err = os.ReadFile(op.Source); errors.Is(err, fs.ErrNotExist) {
		return nil, 0, errors.New("no source available")
	} else if err != nil {
		return nil, 0, fmt.Errorf("source unreadable: %w", err)
	}

	mode, err := resolveMode(op, raw)
	if err != nil {
		return nil, 0, err
	}

	content := raw
	switch {
	case op.Secret:
		content, err = keys.decrypt(string(raw))
	case op.Template:
		content, err = renderTemplate(raw, p.vars, keys.named)
	}

	if err != nil {
		return nil, 0, err
	}

	switch op.Op {
	case OpMerge:
		content, err = mergeContent(existing, content, op.Dest)
	case OpAppend:
		content = slices.Concat(existing, content)
	case OpPrepend:
		content = slices.Concat(content, existing)
	case OpBlock:
		content, err = ensureBlock(existing, content, op.Comment)
	case OpLineInfile:
		content, err = ensureLine(existing, content)
	}

	return content, mode, err
}

func resolveMode(op ResolvedOp, raw []byte) (fs.FileMode, error) {
	switch {
	case op.Secret || op.Template && bytes.Contains(raw, []byte("${secrets.")):
		return 0o600, nil
	case op.Mode != "":
		return internalIO.ParseFileMode(op.Mode)
	}

	if info, err := os.Stat(op.Dest); err == nil && op.Op != OpCopy {
		return info.Mode().Perm(), nil
	}

	return 0o644, nil
}
