package clip

import (
	"github.com/kloudkit/ws-cli/internals/clipboard"
	"github.com/spf13/cobra"
)

var copyCmd = &cobra.Command{
	Use:         "copy",
	Annotations: map[string]string{"since": "next"},
	Short:       "Copy stdin to the clipboard",
	Long:        "Read stdin and write it to the browser clipboard over the workspace IPC socket. Pairs with the pbcopy/xclip/xsel shims for terminal clipboard access.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return clipboard.Copy(cmd.InOrStdin())
	},
}

func init() {
	ClipCmd.AddCommand(copyCmd)
}
