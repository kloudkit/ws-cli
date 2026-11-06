package editor

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kloudkit/ws-cli/cmd/cli"
	"github.com/kloudkit/ws-cli/internals/config"
	editoripc "github.com/kloudkit/ws-cli/internals/editor"
	"github.com/kloudkit/ws-cli/internals/env"
	wsio "github.com/kloudkit/ws-cli/internals/io"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/kloudkit/ws-cli/internals/tempfile"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var errNoEditorOverSSH = errors.New(
	"the editor commands are not available over SSH — there is no editor session to act on",
)

var errDownloadsDisabled = errors.New(
	"file downloads are disabled in this workspace (WS_EDITOR_DISABLE_FILE_DOWNLOADS)",
)

var osExit = os.Exit

var isTerminal = func(r io.Reader) bool {
	file, ok := r.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

var reapLater = func(zip string) error {
	return tempfile.Detach("editor", "reap-download", zip)
}

func exit(code int) error {
	osExit(code)
	return nil
}

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
		if isTerminal(cmd.InOrStdin()) {
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

var downloadCmd = cli.New("download <file>", "next", "Download a file to your computer",
	"Ask the running editor window to download a file of any size to your computer through the browser. The editor asks first, naming the file with its full path and size; once you click Download it returns at once and the transfer shows in the browser's download bar. A folder is zipped into the temp directory first and the zip is downloaded; --zip zips without asking, a terminal is asked, and without either the folder is refused. The zip is deleted at once when the download does not start, and soon after it does, once the editor has opened it. Fails when file downloads are disabled (WS_EDITOR_DISABLE_FILE_DOWNLOADS), when the path is neither a regular file nor a folder, or when the prompt is cancelled or times out. Blocked over SSH.",
	func(cmd *cobra.Command, args []string) error {
		tempfile.Download.Sweep(time.Hour)

		disabled, err := downloadsDisabled()
		if err != nil {
			return err
		}

		if disabled {
			return errDownloadsDisabled
		}

		path, info, err := existingPath(args[0])
		if err != nil {
			return err
		}

		if info.IsDir() {
			return downloadFolder(cmd, path)
		}

		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s: not a regular file", path)
		}

		_, err = sendDownload(cmd, path)

		return err
	})

var reapCmd = cli.New("reap-download <zip>", "next", "Delete a downloaded zip once the editor has it open",
	"Started detached by download once the editor accepts a zip: waits until another process has the zip open, then deletes it, and deletes it anyway after a minute. Refuses any path that is not a ws-cli download zip of this user in the temp directory.",
	func(cmd *cobra.Command, args []string) error {
		return tempfile.Download.Reap(args[0], time.Minute)
	})

var revealCmd = cli.New("reveal <path>", "next", "Reveal a file or folder in the Explorer sidebar",
	"Show and select a workspace file or folder in the editor's Explorer sidebar without opening it, like the Reveal in Explorer View action. Blocked over SSH.",
	func(cmd *cobra.Command, args []string) error {
		path, _, err := existingPath(args[0])
		if err != nil {
			return err
		}

		return editoripc.Reveal(path)
	})

var terminalCmd = cli.New("terminal [-- command...]", "next", "Open a terminal session in the editor",
	"Open a new integrated terminal in the running editor window, in the current directory or --cwd, named with --name. Arguments after -- are joined with spaces and run in the new terminal. Blocked over SSH.",
	func(cmd *cobra.Command, args []string) error {
		cwd, info, err := existingPath(cli.String(cmd, "cwd"))
		if err != nil {
			return err
		}

		if !info.IsDir() {
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
	diagnosticsCmd, downloadCmd, listCmd, notifyCmd, openCmd, reapCmd, revealCmd, selectionCmd, terminalCmd)

func printBody(out io.Writer, body []byte) {
	if len(body) > 0 {
		fmt.Fprintln(out, strings.TrimSpace(string(body)))
	}
}

func downloadsDisabled() (bool, error) {
	value, err := config.Resolve("editor", "disable_file_downloads")
	if err != nil {
		return false, err
	}

	return config.ParseBool(value)
}

func existingPath(value string) (string, os.FileInfo, error) {
	path, err := filepath.Abs(value)
	if err != nil {
		return "", nil, fmt.Errorf("cannot resolve path %q: %w", value, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", path, errors.Unwrap(err))
	}

	return path, info, nil
}

func downloadFolder(cmd *cobra.Command, folder string) error {
	name, ask := filepath.Base(folder), !cli.Bool(cmd, "zip")

	if ask && !isTerminal(cmd.InOrStdin()) {
		styles.PrintWarning(cmd.ErrOrStderr(), fmt.Sprintf("%s is a folder; pass --zip to zip it and download the zip", folder))
		return exit(1)
	}

	file, err := tempfile.Download.Create(name)
	if err != nil {
		return fmt.Errorf("cannot create a zip for %s: %w", folder, err)
	}

	zipped := file.Name()
	release := tempfile.Download.Trap(zipped)
	defer release()

	if ask && !confirm(cmd, fmt.Sprintf("Zip %s into %s and download it? [y/N] ", name, zipped)) {
		_ = file.Close()
		_ = tempfile.Download.Remove(zipped)

		return exit(1)
	}

	err = wsio.ZipFolder(file, folder, zipped)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		_ = tempfile.Download.Remove(zipped)
		return fmt.Errorf("cannot zip %s: %w", folder, err)
	}

	styles.PrintSuccess(cmd.ErrOrStderr(), fmt.Sprintf("Zipped %s into %s", name, zipped))

	downloaded, err := sendDownload(cmd, zipped)
	if !downloaded {
		_ = tempfile.Download.Remove(zipped)
		return err
	}

	if err := reapLater(zipped); err != nil {
		styles.PrintWarning(cmd.ErrOrStderr(), fmt.Sprintf("cannot start the cleanup of %s (%v); it is swept after an hour", zipped, err))
	}

	return nil
}

func sendDownload(cmd *cobra.Command, path string) (bool, error) {
	body, err := editoripc.Download(path)
	if err != nil {
		return false, err
	}

	var result editoripc.DownloadResult

	if cli.Bool(cmd, "raw") {
		printBody(cmd.OutOrStdout(), body)
		_ = json.Unmarshal(body, &result)

		return result.Downloaded, nil
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("error parsing download response: %w", err)
	}

	if !result.Downloaded {
		return false, errors.New("the download was cancelled or timed out in the editor")
	}

	styles.PrintSuccess(cmd.OutOrStdout(), "Download started", styles.Pair{Key: "Path", Value: path})

	return true, nil
}

func confirm(cmd *cobra.Command, question string) bool {
	fmt.Fprint(cmd.ErrOrStderr(), question)

	answer, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')

	return slices.Contains([]string{"y", "yes"}, strings.ToLower(strings.TrimSpace(answer)))
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
	downloadCmd.Flags().Bool("zip", false, "Zip a folder into the temp directory without asking, then download the zip")
	reapCmd.Args, reapCmd.Hidden = cobra.ExactArgs(1), true
	terminalCmd.Flags().String("cwd", ".", "Working directory for the terminal")
	terminalCmd.Flags().String("name", "", "Terminal tab name")
}
