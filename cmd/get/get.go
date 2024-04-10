package get

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var GetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get paths and information of tools",
}

var homeCmd = &cobra.Command{
	Use:   "home",
	Short: "Get the workspace home",
	Run: func(cmd *cobra.Command, args []string) {
		home, exists := os.LookupEnv("WS_ROOT")

		if !exists {
			home = "/workspace"
		}

		fmt.Fprintln(cmd.OutOrStdout(), home)
	},
}

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Get the VSCode settings",
	Run: func(cmd *cobra.Command, args []string) {
		home, exists := os.LookupEnv("HOME")

		if !exists {
			home = "/home/kloud"
		}

		fmt.Fprintln(cmd.OutOrStdout(), home+"/.local/share/code-server/User/settings.json")
	},
}

func init() {
	GetCmd.AddCommand(homeCmd, settingsCmd)
}
