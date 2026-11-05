package editor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/kloudkit/ws-cli/cmd/cli"
	"github.com/kloudkit/ws-cli/internals/config"
	editoripc "github.com/kloudkit/ws-cli/internals/editor"
	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var errNoEditorOverSSH = errors.New(
	"the editor commands are not available over SSH — there is no editor session to act on",
)

var severityStyles = map[string]func(...string) string{
	"Error":       styles.Error.Render,
	"Warning":     styles.Warning.Render,
	"Information": styles.Info.Render,
	"Hint":        styles.Muted.Render,
}

func query(use, short, long string, fetch func(*cobra.Command) ([]byte, error), render func(io.Writer, []byte) error) *cobra.Command {
	return cli.New(use, "0.5.0", short, long, func(cmd *cobra.Command, args []string) error {
		body, err := fetch(cmd)
		if err != nil {
			return err
		}

		if cli.Bool(cmd, "raw") {
			printBody(cmd.OutOrStdout(), body)
			return nil
		}

		return render(cmd.OutOrStdout(), body)
	})
}

var listCmd = query("list", "List the currently open editor tabs",
	"List the editor's open tabs over the IPC socket, with each tab's path, language, and active or dirty state.",
	func(*cobra.Command) ([]byte, error) { return editoripc.FetchEditors() },
	renderEditors)

var selectionCmd = query("selection", "Show the active editor's current selection",
	"Report the active editor's current selection — file, range, and selected text — over the IPC socket. Empty when nothing is selected.",
	func(*cobra.Command) ([]byte, error) { return editoripc.FetchSelection() },
	renderSelection)

var diagnosticsCmd = query("diagnostics", "Show language diagnostics for the workspace (or a single file)",
	"Pull language-server diagnostics (errors, warnings) from the editor over the IPC socket, across the whole workspace or a single file with --uri. Styled table by default, --raw for the JSON.",
	func(cmd *cobra.Command) ([]byte, error) { return editoripc.FetchDiagnostics(cli.String(cmd, "uri")) },
	renderDiagnostics)

var notifyCmd = cli.New("notify", "0.5.0", "Raise a notification in the editor",
	"Read a JSON payload from stdin and raise it as a notification in the running editor window over the workspace IPC socket. Requires a \"message\"; optional \"detail\", \"actions\", \"modal\", \"timeout\", and \"severity\" tune it. Prints the chosen action (or timeout) as JSON. Blocked over SSH.",
	func(cmd *cobra.Command, args []string) error {
		if file, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(file.Fd())) {
			return errors.New(`pipe a JSON payload on stdin, e.g. echo '{"message": "hi"}' | ws editor notify`)
		}

		raw, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("error reading notification payload: %w", err)
		}

		var req editoripc.NotifyRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			return fmt.Errorf("invalid JSON on stdin: %w", err)
		}

		if req.Message == "" {
			return errors.New(`notify requires a JSON payload with a "message"`)
		}

		body, err := editoripc.Notify(req)
		if err != nil {
			return err
		}

		printBody(cmd.OutOrStdout(), body)

		return nil
	})

var openCmd = cli.New("open <file>", "0.5.0", "Open a file in the editor",
	"Open a file in the running editor window over the workspace IPC socket — a tab in the current window by default, a separate one with --new-window, a centered overlay with --modal, jumping to a range with --selection. Fails fast over SSH, where there is no browser editor to open into.",
	func(cmd *cobra.Command, args []string) error {
		path, err := resolvePath(args[0])
		if err != nil {
			return err
		}

		req := editoripc.OpenRequest{Path: path, NewWindow: cli.Bool(cmd, "new-window"), Preview: cli.Bool(cmd, "preview"), Modal: cli.Bool(cmd, "modal")}

		if selection := cli.String(cmd, "selection"); selection != "" {
			if req.Selection, err = parseSelection(selection); err != nil {
				return err
			}
		}

		return editoripc.Open(req)
	})

var downloadCmd = cli.New("download <path>", "next", "Download a workspace file or folder to your computer",
	"Ask the running editor window to download a workspace file or folder to your computer through the browser, like the Explorer's Download… action. Returns once the download has started; its progress shows in the editor and the browser. Fails when file downloads are disabled (WS_EDITOR_DISABLE_FILE_DOWNLOADS) or cancelled in the editor. Blocked over SSH.",
	func(cmd *cobra.Command, args []string) error {
		path, err := existingPath(args[0])
		if err != nil {
			return err
		}

		body, err := editoripc.Download(path)
		if err != nil {
			return err
		}

		if cli.Bool(cmd, "raw") {
			printBody(cmd.OutOrStdout(), body)
			return nil
		}

		var result editoripc.DownloadResult
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("error parsing download response: %w", err)
		}

		if !result.Downloaded {
			return errors.New("the download was cancelled in the editor")
		}

		styles.PrintSuccess(cmd.OutOrStdout(), "Download started", styles.Pair{Key: "Path", Value: path})

		return nil
	})

var revealCmd = cli.New("reveal <path>", "next", "Reveal a file or folder in the Explorer sidebar",
	"Show and select a workspace file or folder in the editor's Explorer sidebar without opening it, like the Reveal in Explorer View action. Blocked over SSH.",
	func(cmd *cobra.Command, args []string) error {
		path, err := existingPath(args[0])
		if err != nil {
			return err
		}

		return editoripc.Reveal(path)
	})

var terminalCmd = cli.New("terminal [-- command...]", "next", "Open a terminal session in the editor",
	"Open a new integrated terminal in the running editor window, in the current directory or --cwd, named with --name. Arguments after -- are joined with spaces and run in the new terminal. Blocked over SSH.",
	func(cmd *cobra.Command, args []string) error {
		cwd, err := existingPath(cli.String(cmd, "cwd"))
		if err != nil {
			return err
		}

		if info, _ := os.Stat(cwd); !info.IsDir() {
			return fmt.Errorf("%s: not a directory", cwd)
		}

		return editoripc.OpenTerminal(editoripc.TerminalRequest{Cwd: cwd, Name: cli.String(cmd, "name"), Command: strings.Join(args, " ")})
	})

var EditorCmd = cli.Group("editor", "0.5.0", "Inspect and drive the active editor session",
	"Query and control the running VS Code / code-server window over the workspace IPC socket — list open tabs, read diagnostics and the current selection, open, reveal, or download a file, or open a terminal. Blocked over SSH, where there is no browser editor to reach.",
	`# List the open editor tabs
ws editor list

# Open a file at a specific line and column
ws editor open src/main.go --selection 12:1

# Preview a document in an overlay instead of a tab
ws editor open README.md --modal

# Run the tests in a new editor terminal
ws editor terminal --name tests -- go test ./...`,
	diagnosticsCmd, downloadCmd, listCmd, notifyCmd, openCmd, revealCmd, selectionCmd, terminalCmd)

func printBody(out io.Writer, body []byte) {
	if len(body) > 0 {
		fmt.Fprintln(out, strings.TrimSpace(string(body)))
	}
}

func existingPath(value string) (string, error) {
	path, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path %q: %w", value, err)
	}

	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("%s: %w", path, errors.Unwrap(err))
	}

	return path, nil
}

func location(p editoripc.Position) string {
	return fmt.Sprintf("%d:%d", p.Line+1, p.Character+1)
}

// resolvePath anchors relative paths to the caller's cwd, since the editor would resolve them against the workspace root.
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
		return editoripc.Position{}, fmt.Errorf("invalid selection %q (want LINE:COL[-LINE:COL], 1-based)", value)
	}

	return editoripc.Position{Line: line - 1, Character: col - 1}, nil
}

func renderEditors(out io.Writer, body []byte) error {
	tabs, err := editoripc.DecodeTabs(body)
	if err != nil {
		return fmt.Errorf("error parsing editor response: %w", err)
	}

	if len(tabs) == 0 {
		styles.PrintEmpty(out, "No open editors")
		return nil
	}

	rows := make([][]string, len(tabs))
	for i, tab := range tabs {
		language := styles.Muted.Render("-")
		if tab.LanguageID != nil {
			language = *tab.LanguageID
		}

		rows[i] = []string{tab.Path, language, styles.Success.Render(styles.Check(tab.Active)), styles.Warning.Render(styles.Check(tab.Dirty))}
	}

	styles.NewTable("Path", "Language", "Active", "Dirty").Title("Open Editors", len(tabs)).Rows(rows...).Print(out)

	return nil
}

func renderSelection(out io.Writer, body []byte) error {
	if len(body) == 0 {
		styles.PrintEmpty(out, "No active selection")
		return nil
	}

	selection, err := editoripc.DecodeSelection(body)
	if err != nil {
		return fmt.Errorf("error parsing selection response: %w", err)
	}

	styles.PrintTitle(out, "Selection")
	styles.PrintKeyValues(out,
		styles.Pair{Key: "Path", Value: selection.Path},
		styles.Pair{Key: "Range", Value: location(selection.Range.Start) + "-" + location(selection.Range.End)},
	)

	if strings.TrimSpace(selection.Text) != "" {
		fmt.Fprintln(out)
		styles.PrintCode(out, selection.Text, selection.Path, selection.Range.Start.Line+1)
	}

	return nil
}

func renderDiagnostics(out io.Writer, body []byte) error {
	var files []editoripc.DiagnosticFile
	if err := json.Unmarshal(body, &files); err != nil {
		return fmt.Errorf("error parsing diagnostics response: %w", err)
	}

	files = slices.DeleteFunc(files, func(f editoripc.DiagnosticFile) bool { return len(f.Items) == 0 })

	total := 0
	for _, file := range files {
		total += len(file.Items)
	}

	if total == 0 {
		styles.PrintEmpty(out, "No diagnostics")
		return nil
	}

	styles.PrintTitle(out, "Diagnostics", total)

	for i, file := range files {
		rows := make([][]string, len(file.Items))
		for j, item := range file.Items {
			severity := item.Severity.Label
			if render, ok := severityStyles[severity]; ok {
				severity = render(severity)
			}

			rows[j] = []string{severity, location(item.Range.Start), item.Source, item.Message}
		}

		if i > 0 {
			fmt.Fprintln(out)
		}

		styles.PrintSubHeader(out, file.URI)
		styles.NewTable("Severity", "Location", "Source", "Message").Rows(rows...).Print(out)
	}

	return nil
}

func init() {
	EditorCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		styles.WrapOutput(cmd)

		if env.IsSSHSession() {
			return errNoEditorOverSSH
		}

		return config.Bootstrap()
	}

	notifyCmd.Example = `# A simple toast
echo '{"message": "Build finished"}' | ws editor notify

# Ask a question and read the chosen action
echo '{"message": "Deploy now?", "actions": ["Yes", "No"]}' | ws editor notify`

	EditorCmd.PersistentFlags().Bool("raw", false, "Output the raw JSON response without styling")
	diagnosticsCmd.Flags().String("uri", "", "Filter to a single file (URI or absolute path)")
	openCmd.Args = cobra.ExactArgs(1)
	openCmd.Flags().Bool("reuse-window", false, "Open in the current window as a tab (default)")
	openCmd.Flags().Bool("new-window", false, "Open in a new window")
	openCmd.Flags().Bool("preview", false, "Open as a preview tab (reuse-window only)")
	openCmd.Flags().Bool("modal", false, "Open in a centered overlay instead of a tab")
	openCmd.Flags().String("selection", "", "Select a range: LINE:COL[-LINE:COL] (1-based)")
	openCmd.MarkFlagsMutuallyExclusive("reuse-window", "new-window", "modal")
	openCmd.MarkFlagsMutuallyExclusive("preview", "modal")
	downloadCmd.Args, revealCmd.Args = cobra.ExactArgs(1), cobra.ExactArgs(1)
	terminalCmd.Flags().String("cwd", ".", "Working directory for the terminal")
	terminalCmd.Flags().String("name", "", "Terminal tab name")
}
