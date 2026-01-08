package info

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

func readJsonFile() map[string]any {
	var content map[string]any

	data, _ := os.ReadFile(config.DefaultManifestPath)

	_ = json.Unmarshal(data, &content)

	return content
}

func readJson(content map[string]any, key string) string {
	keys := strings.Split(key, ".")
	var value any = content

	for _, k := range keys {
		m, ok := value.(map[string]any)
		if !ok {
			return ""
		}

		value = m[k]
	}

	if str, ok := value.(string); ok {
		return str
	}

	return fmt.Sprintf("%v", value)
}

func showVersion(writer io.Writer) {
	content := readJsonFile()

	fmt.Fprintf(writer, "%s\n", styles.Title().Render("Versions"))

	t := styles.Table().Rows(
		[]string{"workspace", readJson(content, "version")},
		[]string{"ws-cli", Version},
		[]string{"VSCode", readJson(content, "vscode.version")},
	)

	fmt.Fprintln(writer, t.Render())
}

var InfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display workspace information",
}

var showVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display installed workspace version",
	Run: func(cmd *cobra.Command, args []string) {
		if all, _ := cmd.Flags().GetBool("all"); all {
			showVersion(cmd.OutOrStdout())
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), Version)
		}
	},
}

func init() {
	showVersionCmd.Flags().Bool("all", false, "Show all version information")

	InfoCmd.AddCommand(showVersionCmd)
}
