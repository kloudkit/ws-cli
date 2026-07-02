package editor

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	editoripc "github.com/kloudkit/ws-cli/internals/editor"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:         "open <file>",
	Annotations: map[string]string{"since": "next"},
	Short:       "Open a file in the editor",
	Long:        "Open a file in the running editor window over the workspace IPC socket — a tab in the current window by default, a separate one with --new-window, jumping to a range with --selection. Fails fast over SSH, where there is no browser editor to open into.",
	Args:        cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		newWindow, _ := cmd.Flags().GetBool("new-window")
		preview, _ := cmd.Flags().GetBool("preview")
		selection, _ := cmd.Flags().GetString("selection")

		path, err := resolvePath(args[0])
		if err != nil {
			return err
		}

		req := editoripc.OpenRequest{
			Path:    path,
			Window:  "reuse",
			Preview: preview,
		}

		if newWindow {
			req.Window = "new"
		}

		if selection != "" {
			parsed, err := parseSelection(selection)
			if err != nil {
				return err
			}

			req.Selection = parsed
		}

		return editoripc.Open(req)
	},
}

// resolvePath makes a filesystem path absolute against the caller's working
// directory so the editor does not resolve a bare or relative path against the
// workspace root. Explicit URIs (containing a scheme) are passed through as-is.
func resolvePath(value string) (string, error) {
	if strings.Contains(value, "://") {
		return value, nil
	}

	abs, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path %q: %w", value, err)
	}

	return abs, nil
}

func parseSelection(value string) (*editoripc.Range, error) {
	start, end, found := strings.Cut(value, "-")

	from, err := parsePosition(start)
	if err != nil {
		return nil, err
	}

	to := from
	if found {
		if to, err = parsePosition(end); err != nil {
			return nil, err
		}
	}

	return &editoripc.Range{Start: from, End: to}, nil
}

func parsePosition(value string) (editoripc.Position, error) {
	rawLine, rawCol, ok := strings.Cut(value, ":")
	line, errLine := strconv.Atoi(rawLine)
	col, errCol := strconv.Atoi(rawCol)

	if !ok || errLine != nil || errCol != nil || line < 1 || col < 1 {
		return editoripc.Position{}, fmt.Errorf(
			"invalid selection %q (want LINE:COL[-LINE:COL], 1-based)", value,
		)
	}

	return editoripc.Position{Line: line - 1, Character: col - 1}, nil
}

func init() {
	openCmd.Flags().Bool("reuse-window", false, "Open in the current window as a tab (default)")
	openCmd.Flags().Bool("new-window", false, "Open in a new window")
	openCmd.Flags().Bool("preview", false, "Open as a preview tab (reuse-window only)")
	openCmd.Flags().String("selection", "", "Select a range: LINE:COL[-LINE:COL] (1-based)")

	openCmd.MarkFlagsMutuallyExclusive("reuse-window", "new-window")

	EditorCmd.AddCommand(openCmd)
}
