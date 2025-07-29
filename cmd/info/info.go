package info

import (
  "bufio"
  "bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func fetchExtensions() string {
  out, _ := exec.Command("code", "--list-extensions", "--show-versions").Output()

	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(out))
	skipHeader := true

	for scanner.Scan() {
		if skipHeader {
			skipHeader = false
			continue
		}

		buf.WriteString("\t\t ")
		buf.WriteString(scanner.Text())
		buf.WriteByte('\n')
	}

	return buf.String()
}

func readJsonFile() map[string]interface{} {
	var content map[string]interface{}

	data, _ := os.ReadFile("/var/lib/workspace/manifest.json")

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
	data, err := os.ReadFile("/var/lib/workspace/state/initialized")
	if err != nil {
		return time.Time{}, 0, err
	}

	parsedTime, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		return time.Time{}, 0, err
	}

	return parsedTime, time.Since(parsedTime), nil
}

var InfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display workspace information",
	Run: func(cmd *cobra.Command, args []string) {
		var content = readJsonFile()
		started, running, _ := readStartup()

		fmt.Fprintln(cmd.OutOrStdout(), "Uptime")
		fmt.Fprintln(cmd.OutOrStdout(), "  started\t", started)
		fmt.Fprintln(cmd.OutOrStdout(), "  running\t", running)
		fmt.Fprintln(cmd.OutOrStdout(), "Versions")
		fmt.Fprintln(cmd.OutOrStdout(), "  workspace\t", readJson(content, "version"))
		fmt.Fprintln(cmd.OutOrStdout(), "  ws-cli\t", Version)
		fmt.Fprintln(cmd.OutOrStdout(), "  VSCode\t", readJson(content, "vscode.version"))
    fmt.Fprintln(cmd.OutOrStdout(), "Extensions")
    fmt.Fprint(cmd.OutOrStdout(), fetchExtensions())
	},
}
