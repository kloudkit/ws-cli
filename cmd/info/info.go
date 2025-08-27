package info

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

func readJsonFile() map[string]any {
	var content map[string]any

	data, _ := os.ReadFile("/var/lib/workspace/manifest.json")

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

	title := styles.HeaderStyle().Render("Versions")

	fmt.Fprintln(writer, title)

	fmt.Fprintf(writer, "  workspace\t%s\n", readJson(content, "version"))
	fmt.Fprintf(writer, "  ws-cli\t%s\n", Version)
	fmt.Fprintf(writer, "  VSCode\t%s\n", readJson(content, "vscode.version"))
}

var InfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display workspace information",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display installed workspace version",
	Run: func(cmd *cobra.Command, args []string) {
		showVersion(cmd.OutOrStdout())
	},
}

func init() {
	InfoCmd.AddCommand(versionCmd)
}
