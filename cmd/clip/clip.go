package clip

import (
	"github.com/kloudkit/ws-cli/cmd/cli"
	"github.com/kloudkit/ws-cli/internals/clipboard"
	"github.com/spf13/cobra"
)

var ClipCmd = cli.Group("clip", "0.2.0", "Interact with the native clipboard",
	"Reach the browser clipboard from the terminal over the workspace IPC socket.",
	`# Save the browser clipboard to a file
ws clip paste > out.txt

# Search within it
ws clip paste | grep "pattern"

# Send command output to the browser clipboard
ls | ws clip copy`,
	cli.New("copy", "0.5.0", "Copy stdin to the clipboard",
		"Read stdin and write it to the browser clipboard over the workspace IPC socket. Pairs with the pbcopy/xclip/xsel shims for terminal clipboard access.",
		func(cmd *cobra.Command, args []string) error { return clipboard.Copy(cmd.InOrStdin()) }),
	cli.New("paste", "0.2.0", "Paste clipboard content",
		"Read the browser clipboard over the workspace IPC socket and write it to stdout — redirect it to a file or pipe it onward. Pairs with the pbcopy/xclip/xsel shims for terminal clipboard access.",
		func(cmd *cobra.Command, args []string) error { return clipboard.Paste(cmd.OutOrStdout()) }),
)
