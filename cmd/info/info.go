package info

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/styles"
)

func showVersion(writer io.Writer) {
	var rows [][]string

	if version := config.WorkspaceVersion(); version != "" {
		rows = append(rows, []string{"workspace", version})
	} else {
		styles.PrintWarning(writer, "Could not determine the workspace version")
	}

	rows = append(rows, []string{"ws-cli", Version()})

	if version, err := config.VSCodeVersion(); err == nil {
		rows = append(rows, []string{"VSCode", version})
	} else {
		styles.PrintWarning(writer, fmt.Sprintf("Could not read the editor version: %v", err))
	}

	fmt.Fprintf(writer, "%s\n", styles.Title().Render("Versions"))
	fmt.Fprintln(writer, styles.Table().Rows(rows...).Render())
}

var InfoCmd = &cobra.Command{
	Use:         "info",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Display workspace information",
	Long:        "Report facts about the running workspace — version, effective environment, installed extensions, live resource metrics, and uptime.",
	Example: `# Show the full version table
ws info version --all

# Watch live resource usage
ws info metrics`,
}

var showVersionCmd = &cobra.Command{
	Use:         "version",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Display installed workspace version",
	Long:        "Print the workspace version. --all expands to the full table — workspace, ws-cli, and VS Code.",
	Run: func(cmd *cobra.Command, args []string) {
		if all, _ := cmd.Flags().GetBool("all"); all {
			showVersion(cmd.OutOrStdout())
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), Version())
		}
	},
}

func init() {
	showVersionCmd.Flags().Bool("all", false, "Show all version information")

	InfoCmd.AddCommand(showVersionCmd)
}
