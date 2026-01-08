package show

import (
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/kloudkit/ws-cli/internals/styles"
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
		homePath := env.String(config.EnvServerRoot, config.DefaultServerRoot)

		if styles.OutputRaw(cmd, homePath) {
			return nil
		}

		styles.PrintTitle(cmd.OutOrStdout(), "Workspace Home Path")
		styles.PrintKeyCode(cmd.OutOrStdout(), "Path", homePath)

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

		if styles.OutputRaw(cmd, settingsPath) {
			return nil
		}

		settingsType := "User"
		if useWorkspace {
			settingsType = "Workspace"
		}

		styles.PrintTitle(cmd.OutOrStdout(), "VS Code Settings Path")
		styles.PrintKeyValue(cmd.OutOrStdout(), "Type", settingsType)
		styles.PrintKeyCode(cmd.OutOrStdout(), "Path", settingsPath)

		return nil
	},
}

func init() {
	pathVscodeCmd.Flags().Bool("workspace", false, "Get the workspace settings")

	pathCmd.AddCommand(pathHomeCmd, pathVscodeCmd)

	ShowCmd.AddCommand(pathCmd)
}
