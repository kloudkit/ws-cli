package cmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/cmd"
	"github.com/kloudkit/ws-cli/internals/config"
	"gotest.tools/v3/assert"
)

const outputEnvFixture = `
envs:
  server:
    properties:
      root:
        type: string
        default: /workspace
        description: Root directory for the workspace.
      port:
        type: integer
        default: 8080
        description: Port on which the web server listens.
`

const helperArgsEnv = "CLI_TEST_HELPER_ARGS"

func TestHelperProcess(t *testing.T) {
	args := os.Getenv(helperArgsEnv)
	if args == "" {
		return
	}

	config.DefaultManifestPath = os.Getenv("CLI_TEST_HELPER_MANIFEST")

	os.Args = append([]string{"ws-cli"}, strings.Split(args, "\x1f")...)
	cmd.RootCmd().SetArgs(os.Args[1:])
	cmd.Execute()
	os.Exit(0)
}

type workspace struct {
	dir  string
	seed string
	env  []string
}

func installWorkspace(t *testing.T) *workspace {
	t.Helper()

	dir := t.TempDir()

	manifest := filepath.Join(dir, "manifest.json")
	assert.NilError(t, os.WriteFile(manifest, []byte(`{"version":"0.5.2"}`), 0o644))

	reference := filepath.Join(dir, "env.reference.yaml")
	assert.NilError(t, os.WriteFile(reference, []byte(outputEnvFixture), 0o644))

	logs := filepath.Join(dir, "logs")
	assert.NilError(t, os.MkdirAll(logs, 0o755))
	assert.NilError(t, os.WriteFile(
		filepath.Join(logs, "workspace.log"),
		[]byte("[2026-10-11T10:00:00.000Z] \x1b[38;2;140;170;238minfo \x1b[m Starting\n[2026-10-11T10:00:01.000Z] error Failed\n"),
		0o644,
	))

	seed := filepath.Join(dir, "seed")
	assert.NilError(t, os.MkdirAll(filepath.Join(seed, "etc"), 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(seed, "etc", "a.conf"), []byte("a"), 0o644))

	home := filepath.Join(dir, "home")
	assert.NilError(t, os.MkdirAll(home, 0o755))

	return &workspace{
		dir:  dir,
		seed: seed,
		env: []string{
			"PATH=" + os.Getenv("PATH"),
			"HOME=" + home,
			"CLI_TEST_HELPER_MANIFEST=" + manifest,
			"WS__INTERNAL_ENV_REFERENCE=" + reference,
			"WS_LOGGING_DIR=" + logs,
			"WS_LOGGING_MAIN_FILE=workspace.log",
		},
	}
}

func (w *workspace) run(t *testing.T, extraEnv []string, args ...string) string {
	t.Helper()

	env := append([]string{helperArgsEnv + "=" + strings.Join(args, "\x1f")}, w.env...)
	env = append(env, extraEnv...)

	command := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	command.Dir = filepath.Join(w.dir, "home")
	command.Env = env

	out, _ := command.CombinedOutput()

	return string(out)
}

var pipedCommands = [][]string{
	{"info", "version", "--all"},
	{"info", "env"},
	{"info", "metrics"},
	{"info", "uptime"},
	{"info", "extensions"},
	{"show", "env", "server.port"},
	{"show", "path", "home"},
	{"show", "ip", "internal"},
	{"template", "list"},
	{"secrets", "generate", "master"},
	{"log", "info", "hello"},
	{"logs"},
}

func commandsWithSeed(seed string) [][]string {
	return append(pipedCommands, []string{"seed", "ls", "--source", seed})
}

func TestPipedOutputIsPlain(t *testing.T) {
	ws := installWorkspace(t)

	for _, args := range commandsWithSeed(ws.seed) {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			t.Parallel()

			out := ws.run(t, nil, args...)

			assert.Assert(t, strings.TrimSpace(out) != "", "no output from %v", args)
			assert.Assert(t, !strings.Contains(out, "\x1b"), "escape in piped output of %v: %q", args, out)
		})
	}
}

func TestClicolorForceColorsPipe(t *testing.T) {
	ws := installWorkspace(t)

	for _, args := range commandsWithSeed(ws.seed) {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			t.Parallel()

			out := ws.run(t, []string{"CLICOLOR_FORCE=1"}, args...)

			assert.Assert(t, strings.Contains(out, "\x1b["), "no color in forced output of %v: %q", args, out)
		})
	}
}

func TestInternalForceColor(t *testing.T) {
	ws := installWorkspace(t)

	t.Run("Truecolor", func(t *testing.T) {
		out := ws.run(t, []string{"WS__INTERNAL_FORCE_COLOR=1"}, "log", "info", "hello")

		assert.Assert(t, strings.Contains(out, "\x1b[38;2;"), "want truecolor, got %q", out)
	})

	t.Run("NoColorWins", func(t *testing.T) {
		out := ws.run(t, []string{"WS__INTERNAL_FORCE_COLOR=1", "NO_COLOR=1"}, "log", "info", "hello")

		assert.Assert(t, !strings.Contains(out, "\x1b"), "want plain, got %q", out)
	})
}

func TestRawStaysPlainWhenForced(t *testing.T) {
	ws := installWorkspace(t)
	force := []string{"CLICOLOR_FORCE=1"}

	assert.Equal(t, ws.run(t, force, "show", "path", "home", "--raw"), "/workspace\n")
	assert.Equal(t, ws.run(t, force, "show", "env", "server.port", "--value"), "8080\n")
}

func TestInvalidLogLevelPrintsOneError(t *testing.T) {
	ws := installWorkspace(t)

	out := ws.run(t, nil, "logs", "--level", "fatal")

	assert.Equal(t, strings.Count(out, "ERROR"), 1, "got %q", out)
	assert.Assert(t, strings.Contains(out, "debug, info, warn, error"), "got %q", out)
}

func TestLogStrip(t *testing.T) {
	ws := installWorkspace(t)

	command := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	command.Dir = filepath.Join(ws.dir, "home")
	command.Env = append([]string{helperArgsEnv + "=log\x1fstrip", "WS__INTERNAL_FORCE_COLOR=1"}, ws.env...)
	command.Stdin = strings.NewReader("\x1b[1;31mred\x1b[0m\n\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\\ncut \x1b[38;2")

	out, err := command.Output()
	assert.NilError(t, err)
	assert.Equal(t, string(out), "red\nlink\ncut ")
}

func TestOutputGolden(t *testing.T) {
	wd, err := os.Getwd()
	assert.NilError(t, err)

	ws := installWorkspace(t)

	cases := map[string][]string{
		"show-env":       {"show", "env", "server.port"},
		"show-path-home": {"show", "path", "home"},
		"template-list":  {"template", "list"},
		"seed-ls":        {"seed", "ls", "--source", ws.seed},
		"log-info":       {"log", "info", "hello", "--indent", "1"},
		"logs-error":     {"logs", "--level", "error"},
		"logs-tail":      {"logs", "--tail", "1"},
		"seed-ls-empty":  {"seed", "ls", "--source", filepath.Join(ws.dir, "home")},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			got := ws.run(t, []string{"WS_SERVER_PORT=8443"}, args...)
			got = strings.ReplaceAll(got, ws.dir, "<tmp>")

			path := filepath.Join(wd, "testdata", "output", name+".txt")

			if os.Getenv("UPDATE_GOLDEN") != "" {
				assert.NilError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				assert.NilError(t, os.WriteFile(path, []byte(got), 0o644))
			}

			want, err := os.ReadFile(path)
			assert.NilError(t, err)
			assert.Equal(t, got, string(want))
		})
	}
}
