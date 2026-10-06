package show

import (
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/spf13/cobra"
)

var pathCmd = &cobra.Command{
	Use:         "path",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Display various paths",
	Long:        "Print well-known workspace paths — the home root or the VS Code settings file.",
}

var pathHomeCmd = makeValueCmd("home", "Display the workspace home path", "Print the workspace home (server root) path.", "Workspace Home Path", "Path", func() (string, error) {
	return config.MustResolve("server", "root"), nil
})

var pathVscodeCmd = &cobra.Command{
	Use:         "vscode-settings",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Display the VS Code settings path",
	Long:        "Print the path to the VS Code settings file — the user file by default, or the folder's with --workspace.",
	RunE: func(cmd *cobra.Command, args []string) error {
		useWorkspace, _ := cmd.Flags().GetBool("workspace")

		settingsPath, settingsType := "/workspace/.vscode/settings.json", "Workspace"

		if !useWorkspace {
			settingsPath = path.GetHomeDirectory("/.local/share/ws-server/User/settings.json")
			settingsType = "User"
		}

		return emit(cmd, "VS Code Settings Path", settingsType, "Path", settingsPath)
	},
}

func init() {
	pathVscodeCmd.Flags().Bool("workspace", false, "Get the workspace settings")

	pathCmd.AddCommand(pathHomeCmd, pathVscodeCmd)

	ShowCmd.AddCommand(pathCmd)
}
