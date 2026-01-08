package show

import (
	"fmt"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/spf13/cobra"
)

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Display various paths",
}

var pathHomeCmd = &cobra.Command{
	Use:   "home",
	Short: "Display the workspace home path",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(
			cmd.OutOrStdout(),
			env.String(config.EnvServerRoot, config.DefaultServerRoot),
		)
		return nil
	},
}

var pathVscodeCmd = &cobra.Command{
	Use:   "vscode-settings",
	Short: "Display the VS Code settings path",
	RunE: func(cmd *cobra.Command, args []string) error {
		useWorkspace, _ := cmd.Flags().GetBool("workspace")

		var settingsPath = "/workspace/.vscode/settings.json"

		if !useWorkspace {
			settingsPath = path.GetHomeDirectory("/.local/share/workspace/User/settings.json")
		}

		fmt.Fprintln(cmd.OutOrStdout(), settingsPath)
		return nil
	},
}

func init() {
	pathVscodeCmd.Flags().Bool("workspace", false, "Get the workspace settings")

	pathCmd.AddCommand(pathHomeCmd, pathVscodeCmd)

	ShowCmd.AddCommand(pathCmd)
}
