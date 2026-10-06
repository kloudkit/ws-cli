package show

import (
	"github.com/kloudkit/ws-cli/internals/net"
	"github.com/spf13/cobra"
)

var ipCmd = &cobra.Command{
	Use:         "ip",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Display IP addresses",
	Long:        "Print the workspace's IP addresses — the internal container address or the node it runs on.",
}

func init() {
	ipCmd.AddCommand(
		makeValueCmd("internal", "Display the internal IP address", "Print the workspace container's internal IP address.", "Internal IP Address", "Address", net.GetInternalIP),
		makeValueCmd("node", "Display the node/host IP address", "Print the IP address of the node hosting the workspace.", "Node IP Address", "Address", net.GetNodeIP),
	)

	ShowCmd.AddCommand(ipCmd)
}
