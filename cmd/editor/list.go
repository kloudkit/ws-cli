package editor

import (
	"fmt"
	"io"

	editoripc "github.com/kloudkit/ws-cli/internals/editor"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:         "list",
	Annotations: map[string]string{"since": "0.5.0"},
	Short:       "List the currently open editor tabs",
	Long:        "List the editor's open tabs over the IPC socket, with each tab's path, language, and active or dirty state.",
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := editoripc.FetchEditors()
		if err != nil {
			return err
		}

		return emit(cmd, body, renderEditors)
	},
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

	styles.PrintTitleCount(out, "Open Editors", len(tabs))

	rows := make([][]string, len(tabs))
	for i, tab := range tabs {
		language := "-"
		if tab.LanguageID != nil {
			language = *tab.LanguageID
		}

		rows[i] = []string{tab.Path, language, styles.Check(tab.Active), styles.Check(tab.Dirty)}
	}

	fmt.Fprintln(out, styles.NewTable("Path", "Language", "Active", "Dirty").Rows(rows...).Render())

	return nil
}

func init() {
	EditorCmd.AddCommand(listCmd)
}
