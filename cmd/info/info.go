package info

import (
	"encoding/json"
	"fmt"
	"os"
  "strings"

	"github.com/spf13/cobra"
)

func readJsonFile() map[string]interface{} {
  var content map[string]interface{}

	data, _ := os.ReadFile("/.manifest")

	_ = json.Unmarshal(data, &content)

  return content
}

func readJson(content map[string]interface{}, key string) string {
  keys := strings.Split(key, ".")
	var value interface{} = content

	for _, k := range keys {
		m, ok := value.(map[string]interface{})
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

var InfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display workspace information",
	Run: func(cmd *cobra.Command, args []string) {
    var content = readJsonFile()

		fmt.Fprintln(cmd.OutOrStdout(), "Versions")
		fmt.Fprintln(cmd.OutOrStdout(), "  workspace\t", readJson(content, "version"))
		fmt.Fprintln(cmd.OutOrStdout(), "  ws-cli\t", Version)
		fmt.Fprintln(cmd.OutOrStdout(), "  VSCode\t", readJson(content, "vscode.version"))
	},
}
