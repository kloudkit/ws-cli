package logs

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/kloudkit/ws-cli/internals/logger"
	"github.com/kloudkit/ws-cli/internals/styles"
)

var validLogLevels = []string{"debug", "info", "warn", "error"}

var validLogTargets = []string{"main", "metrics", "docker", "auth_proxy"}

var LogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Retrieve workspace logs",
	Args:  cobra.NoArgs,
	RunE:  execute,
}

func execute(cmd *cobra.Command, args []string) error {
	follow, _ := cmd.Flags().GetBool("follow")
	tail, _ := cmd.Flags().GetInt("tail")
	level, _ := cmd.Flags().GetString("level")
	target, _ := cmd.Flags().GetString("target")

	if level != "" && !slices.Contains(validLogLevels, level) {
		styles.PrintErrorWithOptions(cmd.ErrOrStderr(), "Invalid log level. Must be one of:", [][]string{
			{"debug", "Debug information"},
			{"info", "General information"},
			{"warn", "Warning messages"},
			{"error", "Error messages only"},
		})
		return fmt.Errorf("invalid log level")
	}

	if !slices.Contains(validLogTargets, target) {
		styles.PrintErrorWithOptions(cmd.ErrOrStderr(), "Invalid log target. Must be one of:", [][]string{
			{"main", "Combined workspace log"},
			{"metrics", "Metrics exporter log"},
			{"docker", "In-container Docker daemon log"},
			{"auth_proxy", "OIDC authentication proxy log"},
		})
		return fmt.Errorf("invalid log target")
	}

	reader, err := logger.NewReader(tail, level, target)
	if err != nil {
		styles.PrintError(cmd.ErrOrStderr(), fmt.Sprintf("Failed to initialize log reader: %s", err))
		return err
	}

	if follow {
		err = reader.FollowLogs(cmd.OutOrStdout())
	} else {
		err = reader.ReadLogs(cmd.OutOrStdout())
	}

	if err != nil {
		styles.PrintError(cmd.ErrOrStderr(), fmt.Sprintf("Error reading logs: %s", err))
		return err
	}

	return nil
}

func init() {
	LogsCmd.Flags().BoolP("follow", "f", false, "Follow log output in real-time")
	LogsCmd.Flags().IntP("tail", "t", 0, "Number of lines to show from the end (0 for all)")
	LogsCmd.Flags().StringP("level", "l", "", "Filter by log level (debug|info|warn|error)")
	LogsCmd.Flags().String("target", "main", "Log target to read (main|metrics|docker|auth_proxy)")
}
