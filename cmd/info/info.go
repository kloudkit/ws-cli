package info

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func readJson(key string) string {
	var pkg map[string]interface{}

	data, _ := os.ReadFile("/.manifest")

	_ = json.Unmarshal(data, &pkg)

	value, _ := pkg[key].(string)

	return value
}

var InfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display workspace information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), "Versions")
		fmt.Fprintln(cmd.OutOrStdout(), "  workspace\t", readJson("version"))
		fmt.Fprintln(cmd.OutOrStdout(), "  ws-cli\t", Version)
		fmt.Fprintln(cmd.OutOrStdout(), "  VSCode\t", readJson("vscode"))
	},
}
