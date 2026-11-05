package testutil

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
)

func StripAnsi(s string) string {
	return ansi.Strip(s)
}

func ResetCommandFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flag.Value.Set(flag.DefValue)
		flag.Changed = false
	})

	for _, c := range cmd.Commands() {
		ResetCommandFlags(c)
	}
}

func StartIPC(t *testing.T, handler http.Handler) {
	t.Helper()

	socket := filepath.Join(t.TempDir(), "pipe.sock")
	t.Setenv("VSCODE_IPC_HOOK_CLI", socket)
	ServeUnix(t, socket, handler)
}

func ServeUnix(t *testing.T, socket string, handler http.Handler) {
	t.Helper()

	listener, err := net.Listen("unix", socket)
	assert.NilError(t, err)

	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()

	t.Cleanup(func() { _ = server.Close() })
}

const fakeEnvReference = `envs:
  editor:
    properties:
      disable_file_downloads:
        type: boolean
        default: false
`

func FakeWorkspace(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	manifest, reference := filepath.Join(dir, "manifest.json"), filepath.Join(dir, "env.reference.yaml")
	assert.NilError(t, os.WriteFile(manifest, []byte(`{"version":"0.0.0"}`), 0o644))
	assert.NilError(t, os.WriteFile(reference, []byte(fakeEnvReference), 0o644))

	previous := config.DefaultManifestPath
	config.DefaultManifestPath = manifest
	t.Cleanup(func() { config.DefaultManifestPath = previous })
	t.Setenv("WS__INTERNAL_ENV_REFERENCE", reference)
}
