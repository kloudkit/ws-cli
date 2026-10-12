package testutil

import (
	"net"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/x/ansi"
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

	listener, err := net.Listen("unix", socket)
	assert.NilError(t, err)

	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()

	t.Cleanup(func() { _ = server.Close() })
}
