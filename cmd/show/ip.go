package show

import (
	"fmt"

	"github.com/kloudkit/ws-cli/internals/net"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var ipCmd = &cobra.Command{
	Use:   "ip",
	Short: "Show IP addresses",
}

var ipInternalCmd = &cobra.Command{
	Use:   "internal",
	Short: "Show internal IP address",
	RunE: func(cmd *cobra.Command, args []string) error {
		ip, err := net.GetInternalIP()

		if err == nil {
			fmt.Fprintln(cmd.OutOrStdout(), styles.Value().Render(ip))
		}

		return err
	},
}

var ipNodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Show node/host IP address",
	RunE: func(cmd *cobra.Command, args []string) error {
		ip, err := net.GetNodeIP()

		if err == nil {
			fmt.Fprintln(cmd.OutOrStdout(), styles.Value().Render(ip))
		}

		return err
	},
}

func init() {
	ipCmd.AddCommand(ipInternalCmd, ipNodeCmd)

	ShowCmd.AddCommand(ipCmd)
}
