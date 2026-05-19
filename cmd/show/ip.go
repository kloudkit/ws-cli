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

func makeIPCmd(use, short, title string, getter func() (string, error)) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			ip, err := getter()
			if err != nil {
				return err
			}

			raw, _ := cmd.Flags().GetBool("raw")
			if styles.OutputRaw(cmd.OutOrStdout(), raw, ip) {
				return nil
			}

			styles.PrintTitle(cmd.OutOrStdout(), title)
			styles.PrintKeyCode(cmd.OutOrStdout(), "Address", ip)

			return nil
		},
	}
}

func init() {
	ipCmd.AddCommand(
		makeIPCmd("internal", "Display the internal IP address", "Internal IP Address", net.GetInternalIP),
		makeIPCmd("node", "Display the node/host IP address", "Node IP Address", net.GetNodeIP),
	)

	ShowCmd.AddCommand(ipCmd)
}
