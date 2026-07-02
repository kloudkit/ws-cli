package clip

import (
	"github.com/kloudkit/ws-cli/internals/clipboard"
	"github.com/spf13/cobra"
)

var pasteCmd = &cobra.Command{
	Use:         "paste",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Paste clipboard content",
	Long:        "Read the browser clipboard over the workspace IPC socket and write it to stdout — redirect it to a file or pipe it onward. Pairs with the pbcopy/xclip/xsel shims for terminal clipboard access.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return clipboard.Paste(cmd.OutOrStdout())
	},
}

func init() {
	ClipCmd.AddCommand(pasteCmd)
}
