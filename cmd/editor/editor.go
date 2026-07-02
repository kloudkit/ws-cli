package editor

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/spf13/cobra"
)

var errNoEditorOverSSH = errors.New(
	"the editor commands are not available over SSH — there is no editor session to act on",
)

var EditorCmd = &cobra.Command{
	Use:         "editor",
	Annotations: map[string]string{"since": "next"},
	Short:       "Inspect and drive the active editor session",
	Long:        "Query and control the running VS Code / code-server window over the workspace IPC socket — list open tabs, read diagnostics and the current selection, or open a file. Blocked over SSH, where there is no browser editor to reach.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if env.IsSSHSession() {
			return errNoEditorOverSSH
		}

		return config.Bootstrap()
	},
}

func emit(cmd *cobra.Command, body []byte, render func(io.Writer, []byte) error) error {
	out := cmd.OutOrStdout()

	if raw, _ := cmd.Flags().GetBool("raw"); raw {
		if len(body) > 0 {
			fmt.Fprintln(out, strings.TrimSpace(string(body)))
		}

		return nil
	}

	return render(out, body)
}

func init() {
	EditorCmd.PersistentFlags().Bool("raw", false, "Output the raw JSON response without styling")
}
