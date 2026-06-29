package editor

import (
	"encoding/json"
	"fmt"
	"io"

	editoripc "github.com/kloudkit/ws-cli/internals/editor"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var selectionCmd = &cobra.Command{
	Use:   "selection",
	Short: "Show the active editor's current selection",
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
		styles.PrintWarning(out, "No active selection")
		return nil
	}

	var selection editoripc.Selection
	if err := json.Unmarshal(body, &selection); err != nil {
		return fmt.Errorf("error parsing selection response: %w", err)
	}

	location := fmt.Sprintf(
		"%d:%d-%d:%d",
		selection.Range.Start.Line+1, selection.Range.Start.Character+1,
		selection.Range.End.Line+1, selection.Range.End.Character+1,
	)

	styles.PrintTitle(out, "Selection")
	styles.PrintKeyValue(out, "Path", selection.Path)
	styles.PrintKeyValue(out, "Range", location)
	styles.PrintKeyCode(out, "Text", selection.Text)

	return nil
}

func init() {
	EditorCmd.AddCommand(selectionCmd)
}
