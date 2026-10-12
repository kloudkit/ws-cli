package editor

import (
	"fmt"
	"io"

	editoripc "github.com/kloudkit/ws-cli/internals/editor"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var selectionCmd = &cobra.Command{
	Use:         "selection",
	Annotations: map[string]string{"since": "0.5.0"},
	Short:       "Show the active editor's current selection",
	Long:        "Report the active editor's current selection — file, range, and selected text — over the IPC socket. Empty when nothing is selected.",
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := editoripc.FetchSelection()
		if err != nil {
			return err
		}

		return emit(cmd, body, renderSelection)
	},
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

	location := fmt.Sprintf(
		"%d:%d-%d:%d",
		selection.Range.Start.Line+1, selection.Range.Start.Character+1,
		selection.Range.End.Line+1, selection.Range.End.Character+1,
	)

	styles.PrintTitle(out, "Selection")
	styles.PrintKeyValues(out,
		styles.Pair{Key: "Path", Value: selection.Path},
		styles.Pair{Key: "Range", Value: location},
		styles.Pair{Key: "Text", Value: selection.Text},
	)

	return nil
}

func init() {
	EditorCmd.AddCommand(selectionCmd)
}
