package get

import (
	"fmt"
	"os"

	"github.com/kloudkit/ws-cli/internals/net"
  "github.com/kloudkit/ws-cli/internals/path"
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
		useWorkspace, _ := cmd.Flags().GetBool("workspace")

		if useWorkspace {
			fmt.Fprintln(cmd.OutOrStdout(), "/workspace/.vscode/settings.json")
			return
		}

		fmt.Fprintln(
      cmd.OutOrStdout(),
      path.GetHomeDirectory("/.local/share/code-server/User/settings.json"),
    )
	},
}

var ipCmd = &cobra.Command{
	Use:   "ip",
	Short: "Get the internal or node IP addresses",
	RunE: func(cmd *cobra.Command, args []string) error {
    var (
      ip string
		  err error
    )

    useNode, _ := cmd.Flags().GetBool("node")

		if useNode {
			ip, err = net.GetNodeIP()
		} else {
			ip, err = net.GetInternalIP()
		}

		if err == nil {
      fmt.Fprintln(cmd.OutOrStdout(), ip)
		}

		return err
	},
}

func init() {
	ipCmd.Flags().Bool("node", false, "Get external node IP address")
	settingsCmd.Flags().Bool("workspace", false, "Get the workspace settings")

	GetCmd.AddCommand(homeCmd, ipCmd, settingsCmd)
}
