package show

import (
	"github.com/kloudkit/ws-cli/internals/net"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var ipCmd = &cobra.Command{
	Use:         "ip",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Display IP addresses",
	Long:        "Print the workspace's IP addresses — the internal container address or the node it runs on.",
}

func makeIPCmd(use, short, long, title string, getter func() (string, error)) *cobra.Command {
	return &cobra.Command{
		Use:         use,
		Annotations: map[string]string{"since": "0.2.0"},
		Short:       short,
		Long:        long,
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
		makeIPCmd("internal", "Display the internal IP address", "Print the workspace container's internal IP address.", "Internal IP Address", net.GetInternalIP),
		makeIPCmd("node", "Display the node/host IP address", "Print the IP address of the node hosting the workspace.", "Node IP Address", net.GetNodeIP),
	)

	ShowCmd.AddCommand(ipCmd)
}
