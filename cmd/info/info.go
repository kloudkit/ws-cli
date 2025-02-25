package info

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
  "time"

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

func readStartup() (time.Time, time.Duration, error) {
	data, err := os.ReadFile("/.startup")
	if err != nil {
		return time.Time{}, 0, err
	}

	parsedTime, err := time.Parse(time.ANSIC, strings.TrimSpace(string(data)))
	if err != nil {
		return time.Time{}, 0, err
	}

	return parsedTime, time.Now().Sub(parsedTime), nil
}

var InfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display workspace information",
	Run: func(cmd *cobra.Command, args []string) {
		var content = readJsonFile()
    started, running, _ := readStartup()

		fmt.Fprintln(cmd.OutOrStdout(), "Versions")
		fmt.Fprintln(cmd.OutOrStdout(), "  workspace\t", readJson(content, "version"))
		fmt.Fprintln(cmd.OutOrStdout(), "  ws-cli\t", Version)
		fmt.Fprintln(cmd.OutOrStdout(), "  VSCode\t", readJson(content, "vscode.version"))
		fmt.Fprintln(cmd.OutOrStdout(), "Uptime")
		fmt.Fprintln(cmd.OutOrStdout(), "  started\t", started)
		fmt.Fprintln(cmd.OutOrStdout(), "  running\t", running)
	},
}
