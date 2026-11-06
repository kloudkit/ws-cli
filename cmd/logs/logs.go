package logs

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kloudkit/ws-cli/cmd/cli"
	"github.com/kloudkit/ws-cli/internals/logger"
	"github.com/spf13/cobra"
)

var (
	validLogLevels  = []string{"debug", "info", "warn", "error"}
	validLogTargets = []string{"main", "metrics", "docker", "cloudflared"}
)

var LogsCmd = cli.New("logs", "0.2.0", "Retrieve workspace logs",
	"Read a workspace daemon's log: the main log by default, or --target metrics|docker|cloudflared. Filter by --level, limit with --tail, or stream live with --follow.",
	func(cmd *cobra.Command, args []string) error {
		level, target := cli.String(cmd, "level"), cli.String(cmd, "target")

		if level != "" && !slices.Contains(validLogLevels, level) {
			return fmt.Errorf("invalid log level %q (valid: %s)", level, strings.Join(validLogLevels, ", "))
		}

		if !slices.Contains(validLogTargets, target) {
			return fmt.Errorf("invalid log target %q (valid: %s)", target, strings.Join(validLogTargets, ", "))
		}

		reader, err := logger.NewReader(cli.Int(cmd, "tail"), level, target)
		if err != nil {
			return err
		}

		return reader.ReadLogs(cmd.OutOrStdout(), cli.Bool(cmd, "follow"))
	})

func init() {
	LogsCmd.Args = cobra.NoArgs
	LogsCmd.Example = `# Stream errors live, starting from the last 100 lines
ws-cli logs --level=error --tail=100 --follow

# Read a specific target instead of the main log
ws-cli logs --target metrics`

	LogsCmd.Flags().BoolP("follow", "f", false, "Follow log output in real-time")
	LogsCmd.Flags().IntP("tail", "t", 0, "Number of lines to show from the end (0 for all)")
	LogsCmd.Flags().StringP("level", "l", "", "Filter by log level (debug|info|warn|error)")
	LogsCmd.Flags().String("target", "main", "Log target to read (main|metrics|docker|cloudflared)")
}
