package clipboard

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/kloudkit/ws-cli/internals/net"
	"github.com/spf13/cobra"
)

var ClipboardCmd = &cobra.Command{
	Use:     "clipboard",
	Aliases: []string{"clip"},
	Short:   "Interact with the native clipboard",
}

var copyCmd = &cobra.Command{
	Use:     "copy",
	Aliases: []string{"cp"},
	Args:    cobra.NoArgs,
	Short:   "Copy piped content to the clipboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		piped, err := io.ReadAll(os.Stdin)

		if err != nil {
			return fmt.Errorf("failed to read from stdin: %v", err)
		}

		client := net.GetIPCClient()

		req, err := http.NewRequest("POST", "http://localhost/clipboard", bytes.NewReader(piped))
		if err != nil {
			return fmt.Errorf("error sending to workspace socket: %v", err)
		}

		req.Header.Set("Content-Type", "text/plain")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("error making request: %v", err)
		}
		resp.Body.Close()

		return nil
	},
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
	ClipboardCmd.AddCommand(copyCmd, pasteCmd)
}
