package editor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	editoripc "github.com/kloudkit/ws-cli/internals/editor"
	"github.com/spf13/cobra"
)

var notifyCmd = &cobra.Command{
	Use:         "notify",
	Annotations: map[string]string{"since": "next"},
	Short:       "Raise a notification in the editor",
	Long:        "Read a JSON payload from stdin and raise it as a notification in the running editor window over the workspace IPC socket. Requires a \"message\"; optional \"detail\", \"actions\", \"modal\", \"timeout\", and \"severity\" tune it. Prints the chosen action (or timeout) as JSON. Blocked over SSH.",
	Example: `# A simple toast
echo '{"message": "Build finished"}' | ws editor notify

# Ask a question and read the chosen action
echo '{"message": "Deploy now?", "actions": ["Yes", "No"]}' | ws editor notify`,
	RunE: func(cmd *cobra.Command, args []string) error {
		raw, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("error reading notification payload: %w", err)
		}

		var req editoripc.NotifyRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			return fmt.Errorf("invalid JSON on stdin: %w", err)
		}

		if req.Message == "" {
			return errors.New(`notify requires a JSON payload with a "message"`)
		}

		body, err := editoripc.Notify(req)
		if err != nil {
			return err
		}

		if len(body) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(body)))
		}

		return nil
	},
}

func init() {
	EditorCmd.AddCommand(notifyCmd)
}
