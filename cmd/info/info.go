package info

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func vscodeVersion() string {
	var pkg struct{ Version string }

	data, _ := os.ReadFile("/usr/lib/code-server/lib/vscode/package.json")

	_ = json.Unmarshal(data, &pkg)

	return pkg.Version
}

func workspaceVersion() string {
	data, _ := os.ReadFile("/.version")

	return strings.TrimSpace(string(data))
}

var InfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display workspace information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), "Versions")
		fmt.Fprintln(cmd.OutOrStdout(), "  workspace\t", workspaceVersion())
		fmt.Fprintln(cmd.OutOrStdout(), "  ws-cli\t", Version)
		fmt.Fprintln(cmd.OutOrStdout(), "  VSCode\t", vscodeVersion())
	},
}
