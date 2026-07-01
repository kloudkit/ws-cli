package editor

import (
	"encoding/json"
	"fmt"
	"io"

	editoripc "github.com/kloudkit/ws-cli/internals/editor"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the currently open editor tabs",
	Long:  "List the editor's open tabs over the IPC socket, with each tab's path, language, and active or dirty state.",
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := editoripc.FetchEditors()
		if err != nil {
			return err
		}

		return emit(cmd, body, renderEditors)
	},
}

func renderEditors(out io.Writer, body []byte) error {
	var tabs []editoripc.Tab
	if err := json.Unmarshal(body, &tabs); err != nil {
		return fmt.Errorf("error parsing editor response: %w", err)
	}

	if len(tabs) == 0 {
		styles.PrintWarning(out, "No open editors")
		return nil
	}

	fmt.Fprintf(out, "%s\n", styles.TitleWithCount("Open Editors", len(tabs)))

	rows := make([][]string, len(tabs))
	for i, tab := range tabs {
		language := "-"
		if tab.LanguageID != nil {
			language = *tab.LanguageID
		}

		rows[i] = []string{tab.Path, language, check(tab.Active), check(tab.Dirty)}
	}

	fmt.Fprintf(out, "%s\n", styles.Table("Path", "Language", "Active", "Dirty").Rows(rows...).Render())

	return nil
}

func check(value bool) string {
	if value {
		return "✓"
	}

	return ""
}

func init() {
	EditorCmd.AddCommand(listCmd)
}
