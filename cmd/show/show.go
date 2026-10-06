package show

import (
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var ShowCmd = &cobra.Command{
	Use:         "show",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Display information about the current workspace instance",
	Long:        "Resolve and print facts about this workspace instance — settings, IP addresses, and paths. --raw drops the styling for use in scripts.",
	Example: `# Resolve a setting by its dotted key
ws show env server.port

# Reverse-tunnel a local port to the workspace node
ws_node_ip=$(ws show ip node); ssh -N -R "3001:${ws_node_ip}:3001" "${ws_node_ip}"`,
}

func makeValueCmd(use, short, long, title, label string, getter func() (string, error)) *cobra.Command {
	return &cobra.Command{
		Use:         use,
		Annotations: map[string]string{"since": "0.2.0"},
		Short:       short,
		Long:        long,
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := getter()
			if err != nil {
				return err
			}

			return emit(cmd, title, "", label, value)
		},
	}
}

func emit(cmd *cobra.Command, title, kind, label, value string) error {
	out := cmd.OutOrStdout()

	raw, _ := cmd.Flags().GetBool("raw")
	if styles.OutputRaw(out, raw, value) {
		return nil
	}

	styles.PrintTitle(out, title)
	if kind != "" {
		styles.PrintKeyValue(out, "Type", kind)
	}
	styles.PrintKeyCode(out, label, value)

	return nil
}

func init() {
	ShowCmd.PersistentFlags().Bool("raw", false, "Output raw value without styling")
}
