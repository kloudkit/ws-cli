package seed

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/env"
	internalIO "github.com/kloudkit/ws-cli/internals/io"
	"github.com/kloudkit/ws-cli/internals/path"
)

type Vars struct {
	Home       string
	User       string
	ServerRoot string
}

type ResolvedOp struct {
	Dest     string
	Source   string
	Content  *string
	Mode     string
	Secret   bool
	Op       Op
	Template bool
	Force    bool
	Comment  string
}

type Plan struct {
	Source   string
	Vars     Vars
	Manifest *Manifest
	Ops      []ResolvedOp
}

func resolveVars() Vars {
	username := ""
	if u, err := user.Current(); err == nil {
		username = u.Username
	}

	serverRoot, _ := config.Resolve("server", "root")

	return Vars{
		Home:       env.Home(),
		User:       username,
		ServerRoot: serverRoot,
	}
}

func (v Vars) expand(value string) (string, error) {
	replaced := strings.NewReplacer(
		"${ws_home}", v.Home,
		"${ws_user}", v.User,
		"${ws_server_root}", v.ServerRoot,
	).Replace(value)

	return path.Expand(replaced)
}

func ResolveSource(flag string) (string, error) {
	if flag == "" {
		resolved, err := config.Resolve("seed", "source")
		if err != nil {
			return "", err
		}

		flag = resolved
	}

	return path.Expand(flag)
}

func BuildPlan(source string, force bool) (*Plan, error) {
	vars := resolveVars()

	var manifest *Manifest
	if manifestPath := ManifestPath(source); internalIO.FileExists(manifestPath) {
		loaded, err := LoadManifest(manifestPath)
		if err != nil {
			return nil, err
		}

		manifest = loaded
	}

	ops, err := buildPlan(source, manifest, vars, force)
	if err != nil {
		return nil, err
	}

	return &Plan{Source: source, Vars: vars, Manifest: manifest, Ops: ops}, nil
}

func buildPlan(source string, manifest *Manifest, vars Vars, force bool) ([]ResolvedOp, error) {
	plan := map[string]ResolvedOp{}

	mirror, err := walkMirror(source)
	if err != nil {
		return nil, err
	}

	for dest, src := range mirror {
		plan[dest] = ResolvedOp{Dest: dest, Source: src, Op: OpCopy, Force: force}
	}

	if manifest != nil {
		for rawDest, op := range manifest.Seeds {
			dest, err := vars.expand(rawDest)
			if err != nil {
				return nil, fmt.Errorf("seed %q: %w", rawDest, err)
			}

			resolved := ResolvedOp{
				Dest:     dest,
				Content:  op.Content,
				Mode:     op.Mode,
				Secret:   op.Secret,
				Op:       op.Op,
				Template: op.Template,
				Force:    force || op.Force,
				Comment:  op.Comment,
			}

			if op.Content == nil {
				resolved.Source = rhymingSource(source, dest)
			}

			plan[dest] = resolved
		}
	}

	dests := make([]string, 0, len(plan))
	for dest := range plan {
		dests = append(dests, dest)
	}
	sort.Strings(dests)

	ops := make([]ResolvedOp, 0, len(plan))
	for _, dest := range dests {
		ops = append(ops, plan[dest])
	}

	return ops, nil
}

func rhymingSource(source, dest string) string {
	return filepath.Join(source, strings.TrimPrefix(dest, string(os.PathSeparator)))
}

func (p *Plan) filterDests(args []string) ([]ResolvedOp, error) {
	wanted := map[string]bool{}
	for _, arg := range args {
		resolved, err := p.Vars.expand(arg)
		if err != nil {
			return nil, err
		}

		wanted[resolved] = true
	}

	filtered := make([]ResolvedOp, 0, len(args))
	for _, op := range p.Ops {
		if wanted[op.Dest] {
			filtered = append(filtered, op)
			delete(wanted, op.Dest)
		}
	}

	if len(wanted) > 0 {
		missing := make([]string, 0, len(wanted))
		for dest := range wanted {
			missing = append(missing, dest)
		}
		sort.Strings(missing)

		return nil, fmt.Errorf("no seed entry for: %s", strings.Join(missing, ", "))
	}

	return filtered, nil
}
