package clip

import (
	"github.com/kloudkit/ws-cli/internals/clipboard"
	"github.com/spf13/cobra"
)

var pasteCmd = &cobra.Command{
	Use:   "paste",
	Short: "Paste clipboard content",
	RunE: func(cmd *cobra.Command, args []string) error {
		return clipboard.Paste(cmd.OutOrStdout())
	},
}

func init() {
	ClipCmd.AddCommand(pasteCmd)
}
