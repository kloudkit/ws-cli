package seed

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/env"
	internalIO "github.com/kloudkit/ws-cli/internals/io"
	"github.com/kloudkit/ws-cli/internals/path"
)

var tokenRe = regexp.MustCompile(`\$\{([^}]*)\}`)

type Vars struct {
	Home       string
	User       string
	ServerRoot string
}

type ResolvedOp struct {
	SeedOp
	Dest   string
	Source string
}

type Plan struct {
	source   string
	vars     Vars
	manifest *Manifest
	Ops      []ResolvedOp
}

func resolveVars() Vars {
	vars := Vars{Home: env.Home()}
	if u, err := user.Current(); err == nil {
		vars.User = u.Username
	}

	vars.ServerRoot, _ = config.Resolve("server", "root")

	return vars
}

func (v Vars) tokens() map[string]string {
	return map[string]string{"ws_home": v.Home, "ws_user": v.User, "ws_server_root": v.ServerRoot}
}

func (v Vars) expand(value string) string {
	for name, replacement := range v.tokens() {
		value = strings.ReplaceAll(value, "${"+name+"}", replacement)
	}

	return path.Expand(value)
}

func renderTemplate(content []byte, vars Vars, secret func(string) ([]byte, error)) ([]byte, error) {
	var failure error

	rendered := tokenRe.ReplaceAllFunc(content, func(match []byte) []byte {
		if failure != nil {
			return nil
		}

		token := string(match[2 : len(match)-1])
		if value, ok := vars.tokens()[token]; ok {
			return []byte(value)
		}

		name, ok := strings.CutPrefix(token, "secrets.")
		if !ok {
			failure = fmt.Errorf("unknown template token ${%s}", token)
			return nil
		}

		value, err := secret(name)
		failure = err

		return value
	})

	if failure != nil {
		return nil, failure
	}

	return rendered, nil
}

func ResolveSource(flag string, explicit bool) (string, error) {
	if flag == "" {
		resolved, err := config.Resolve("seed", "source")
		if err != nil {
			return "", err
		}

		flag = resolved
	}

	source := path.Expand(flag)

	if explicit {
		info, err := os.Stat(source)
		if err != nil {
			return "", fmt.Errorf("seed source [%s] not found", source)
		}

		if !info.IsDir() {
			return "", fmt.Errorf("seed source [%s] is not a directory", source)
		}
	}

	return source, nil
}

func BuildPlan(source string, force bool) (*Plan, error) {
	plan := &Plan{source: source, vars: resolveVars(), manifest: &Manifest{}}

	if data, err := os.ReadFile(manifestPath(source)); err == nil {
		if plan.manifest, err = parseManifest(data); err != nil {
			return nil, err
		}
	} else if internalIO.FileExists(manifestPath(source)) {
		return nil, fmt.Errorf("failed to read manifest %q: %w", manifestPath(source), err)
	}

	byDest := map[string]ResolvedOp{}

	if info, err := os.Stat(source); err == nil && info.IsDir() {
		err := filepath.WalkDir(source, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}

			rel, err := filepath.Rel(source, p)
			if err == nil && rel != manifestName {
				dest := string(os.PathSeparator) + rel
				byDest[dest] = ResolvedOp{SeedOp: SeedOp{Op: OpCopy, Force: force}, Dest: dest, Source: p}
			}

			return err
		})
		if err != nil {
			return nil, err
		}
	}

	for rawDest, op := range plan.manifest.Seeds {
		dest := plan.vars.expand(rawDest)
		op.Force = force || op.Force
		resolved := ResolvedOp{SeedOp: op, Dest: dest}

		if op.Content == nil {
			resolved.Source = rhymingSource(source, dest)
		}

		byDest[dest] = resolved
	}

	for _, dest := range slices.Sorted(maps.Keys(byDest)) {
		plan.Ops = append(plan.Ops, byDest[dest])
	}

	return plan, nil
}

func rhymingSource(source, dest string) string {
	return filepath.Join(source, strings.TrimPrefix(dest, string(os.PathSeparator)))
}

func (p *Plan) filterDests(args []string) ([]ResolvedOp, error) {
	wanted := map[string]bool{}
	for _, arg := range args {
		wanted[p.vars.expand(arg)] = true
	}

	var filtered []ResolvedOp
	for _, op := range p.Ops {
		if wanted[op.Dest] {
			filtered = append(filtered, op)
			delete(wanted, op.Dest)
		}
	}

	if len(wanted) > 0 {
		return nil, fmt.Errorf("no seed entry for: %s", strings.Join(slices.Sorted(maps.Keys(wanted)), ", "))
	}

	return filtered, nil
}
