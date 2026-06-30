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

type Options struct {
	Source    string
	Force     bool
	Dests     []string
	MasterKey string
	Out       io.Writer
	Styled    bool
}

type reporter struct {
	out    io.Writer
	styled bool
}

func (r reporter) seeded(dest string) {
	if r.styled {
		styles.PrintSuccess(r.out, fmt.Sprintf("Seeded [%s]", dest))
		return
	}

	fmt.Fprintf(r.out, "Seeded [%s]\n", dest)
}

func (r reporter) skip(dest, reason string) {
	if r.styled {
		styles.PrintWarning(r.out, fmt.Sprintf("Skipping [%s] (%s)", dest, reason))
		return
	}

	fmt.Fprintf(r.out, "Skipping [%s] (%s)\n", dest, reason)
}

func (r reporter) notice(dest string) {
	message := fmt.Sprintf("[%s] runs next boot; ensure +x if executable", dest)

	if r.styled {
		styles.PrintKeyValue(r.out, "Notice", message)
		return
	}

	fmt.Fprintf(r.out, "Notice %s\n", message)
}

type keyResolver struct {
	flag    string
	secrets map[string]string
	key     []byte
	loaded  bool
	err     error
}

func (k *keyResolver) master() ([]byte, error) {
	if !k.loaded {
		k.key, k.err = secrets.ResolveMasterKey(k.flag)
		k.loaded = true
	}

	return k.key, k.err
}

func (k *keyResolver) zero() {
	for i := range k.key {
		k.key[i] = 0
	}
}

func (k *keyResolver) resolveNamed(name string) ([]byte, error) {
	value, ok := k.secrets[name]
	if !ok {
		return nil, fmt.Errorf("secret %q not declared", name)
	}

	master, err := k.master()
	if err != nil {
		return nil, err
	}

	resolved, err := secrets.ResolveEncryptedValue(value)
	if err != nil {
		return nil, err
	}

	return secrets.Decrypt(secrets.NormalizeEncrypted(resolved), master)
}

func Apply(opts Options) error {
	plan, err := BuildPlan(opts.Source, opts.Force)
	if err != nil {
		return err
	}

	ops := plan.Ops
	if len(opts.Dests) > 0 {
		ops, err = plan.filterDests(opts.Dests)
		if err != nil {
			return err
		}
	}

	declared := map[string]string{}
	if plan.Manifest != nil {
		declared = plan.Manifest.Secrets
	}

	keys := &keyResolver{flag: opts.MasterKey, secrets: declared}
	defer keys.zero()
	rep := reporter{out: opts.Out, styled: opts.Styled}

	failures := 0
	for _, op := range ops {
		if err := plan.applyOne(op, keys, rep); err != nil {
			failures++
		}
	}

	if failures > 0 {
		noun := "entries"
		if failures == 1 {
			noun = "entry"
		}

		return fmt.Errorf("%d seed %s failed to apply", failures, noun)
	}

	return nil
}

func (p *Plan) applyOne(op ResolvedOp, keys *keyResolver, rep reporter) error {
	ancestor := nearestExistingAncestor(op.Dest)
	if !ownsPath(ancestor) {
		rep.skip(op.Dest, "destination not owned")
		return fmt.Errorf("destination not owned")
	}

	if op.Op != OpBlock && !internalIO.CanOverride(op.Dest, op.Force) {
		rep.skip(op.Dest, "exists")
		return nil
	}

	content, mode, err := p.materialize(op, keys)
	if err != nil {
		rep.skip(op.Dest, err.Error())
		return err
	}

	if op.Op == OpBlock && bytes.Equal(content, readExisting(op.Dest)) {
		rep.seeded(op.Dest)
		return nil
	}

	anchor := chooseAnchor(op.Dest, p.Vars, ancestor)
	if err := writeAtomic(anchor, op.Dest, content, mode); err != nil {
		rep.skip(op.Dest, err.Error())
		return err
	}

	if consumedNotice(op.Dest, p.Vars.Home) {
		rep.notice(op.Dest)
	}

	rep.seeded(op.Dest)
	return nil
}

func (p *Plan) materialize(op ResolvedOp, keys *keyResolver) ([]byte, fs.FileMode, error) {
	raw, err := p.sourceBytes(op)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, 0, fmt.Errorf("no source available")
		}

		return nil, 0, fmt.Errorf("source unreadable: %w", err)
	}

	mode, err := resolveMode(op, op.Secret || (op.Template && referencesSecrets(raw)))
	if err != nil {
		return nil, 0, err
	}

	content, err := p.transform(op, raw, keys)
	if err != nil {
		return nil, 0, err
	}

	switch op.Op {
	case OpMerge:
		if content, err = mergeContent(readExisting(op.Dest), content, op.Dest); err != nil {
			return nil, 0, err
		}
	case OpAppend:
		content = slices.Concat(readExisting(op.Dest), content)
	case OpPrepend:
		content = slices.Concat(content, readExisting(op.Dest))
	case OpBlock:
		if content, err = ensureBlock(readExisting(op.Dest), content, op.Comment); err != nil {
			return nil, 0, err
		}
	}

	return content, mode, nil
}

func (p *Plan) transform(op ResolvedOp, raw []byte, keys *keyResolver) ([]byte, error) {
	if op.Secret {
		resolved, err := secrets.ResolveEncryptedValue(string(raw))
		if err != nil {
			return nil, fmt.Errorf("secret source unresolved")
		}

		master, err := keys.master()
		if err != nil {
			return nil, fmt.Errorf("master key unavailable")
		}

		plain, err := secrets.Decrypt(secrets.NormalizeEncrypted(resolved), master)
		if err != nil {
			return nil, fmt.Errorf("decrypt failed")
		}

		return plain, nil
	}

	if op.Template {
		return renderTemplate(raw, p.Vars, keys.resolveNamed)
	}

	return raw, nil
}

func (p *Plan) sourceBytes(op ResolvedOp) ([]byte, error) {
	if op.Content != nil {
		return []byte(*op.Content), nil
	}

	return os.ReadFile(op.Source)
}

func resolveMode(op ResolvedOp, secretBearing bool) (fs.FileMode, error) {
	if secretBearing {
		return 0o600, nil
	}

	if op.Mode != "" {
		return internalIO.ParseFileMode(op.Mode)
	}

	if op.Op.inPlace() {
		if info, err := os.Stat(op.Dest); err == nil {
			return info.Mode().Perm(), nil
		}
	}

	return 0o644, nil
}

func readExisting(dest string) []byte {
	data, err := os.ReadFile(dest)
	if err != nil {
		return nil
	}

	return data
}
