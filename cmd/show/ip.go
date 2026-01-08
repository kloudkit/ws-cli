package show

import (
	"github.com/kloudkit/ws-cli/internals/net"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var ipCmd = &cobra.Command{
	Use:   "ip",
	Short: "Display IP addresses",
}

var ipInternalCmd = &cobra.Command{
	Use:   "internal",
	Short: "Display the internal IP address",
	RunE: func(cmd *cobra.Command, args []string) error {
		ip, err := net.GetInternalIP()
		if err != nil {
			return err
		}

		if styles.OutputRaw(cmd, ip) {
			return nil
		}

		styles.PrintTitle(cmd.OutOrStdout(), "Internal IP Address")
		styles.PrintKeyCode(cmd.OutOrStdout(), "Address", ip)

		return nil
	},
}

var ipNodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Display the node/host IP address",
	RunE: func(cmd *cobra.Command, args []string) error {
		ip, err := net.GetNodeIP()
		if err != nil {
			return err
		}

		if styles.OutputRaw(cmd, ip) {
			return nil
		}

		styles.PrintTitle(cmd.OutOrStdout(), "Node IP Address")
		styles.PrintKeyCode(cmd.OutOrStdout(), "Address", ip)

		return nil
	},
}

func init() {
	ipCmd.AddCommand(ipInternalCmd, ipNodeCmd)

	ShowCmd.AddCommand(ipCmd)
}
