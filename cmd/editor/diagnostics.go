package editor

import (
	"encoding/json"
	"fmt"
	"io"

	editoripc "github.com/kloudkit/ws-cli/internals/editor"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var diagnosticsCmd = &cobra.Command{
	Use:   "diagnostics",
	Short: "Show language diagnostics for the workspace (or a single file)",
	RunE: func(cmd *cobra.Command, args []string) error {
		uri, _ := cmd.Flags().GetString("uri")

		body, err := editoripc.FetchDiagnostics(uri)
		if err != nil {
			return err
		}

		return emit(cmd, body, renderDiagnostics)
	},
}

func renderDiagnostics(out io.Writer, body []byte) error {
	var files []editoripc.DiagnosticFile
	if err := json.Unmarshal(body, &files); err != nil {
		return fmt.Errorf("error parsing diagnostics response: %w", err)
	}

	total := 0
	for _, file := range files {
		total += len(file.Items)
	}

	if total == 0 {
		styles.PrintSuccess(out, "No diagnostics")
		return nil
	}

	fmt.Fprintf(out, "%s\n", styles.TitleWithCount("Diagnostics", total))

	for _, file := range files {
		if len(file.Items) == 0 {
			continue
		}

		fmt.Fprintf(out, "%s\n", styles.SubHeader().Render(file.URI))

		rows := make([][]string, len(file.Items))
		for i, item := range file.Items {
			location := fmt.Sprintf("%d:%d", item.Range.Start.Line+1, item.Range.Start.Character+1)
			rows[i] = []string{item.Severity.Label, location, item.Source, item.Message}
		}

		fmt.Fprintf(out, "%s\n", styles.Table("Severity", "Location", "Source", "Message").Rows(rows...).Render())
	}

	return nil
}

func init() {
	diagnosticsCmd.Flags().String("uri", "", "Filter to a single file (URI or absolute path)")

	EditorCmd.AddCommand(diagnosticsCmd)
}
