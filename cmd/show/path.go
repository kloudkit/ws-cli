package show

import (
	"fmt"

	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show various paths",
}

var pathHomeCmd = &cobra.Command{
	Use:   "home",
	Short: "Show workspace home path",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(
			cmd.OutOrStdout(),
			styles.Value().Render(env.String("WS_SERVER_ROOT", "/workspace")),
		)
	},
}

var pathVscodeCmd = &cobra.Command{
	Use:   "vscode-settings",
	Short: "Show VS Code settings path",
	Run: func(cmd *cobra.Command, args []string) {
		useWorkspace, _ := cmd.Flags().GetBool("workspace")

		var settingsPath = "/workspace/.vscode/settings.json"

		if !useWorkspace {
			settingsPath = path.GetHomeDirectory("/.local/share/code-server/User/settings.json")
		}

		fmt.Fprintln(cmd.OutOrStdout(), styles.Value().Render(settingsPath))
	},
}

func init() {
	pathVscodeCmd.Flags().Bool("workspace", false, "Get the workspace settings")

	pathCmd.AddCommand(pathHomeCmd, pathVscodeCmd)

	ShowCmd.AddCommand(pathCmd)
}
