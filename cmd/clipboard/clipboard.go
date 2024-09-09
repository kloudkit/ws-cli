package clipboard

import (
	"fmt"
	"io"

	"github.com/kloudkit/ws-cli/internals/net"
	"github.com/spf13/cobra"
)

var ClipboardCmd = &cobra.Command{
	Use:     "clipboard",
	Aliases: []string{"clip"},
	Short:   "Interact with the native clipboard",
}

var pasteCmd = &cobra.Command{
	Use:   "paste",
	Short: "Paste clipboard content to the terminal",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := net.GetIPCClient()

		resp, err := client.Get("http://localhost/clipboard")
		if err != nil {
			return fmt.Errorf("error retrieving from workspace socket: %v", err)
		}
		defer resp.Body.Close()

		_, err = io.Copy(cmd.OutOrStdout(), resp.Body)
		if err != nil {
			return fmt.Errorf("error outputting clipboard data: %v", err)
		}

		return nil
	},
}

func init() {
	ClipboardCmd.AddCommand(pasteCmd)
}
