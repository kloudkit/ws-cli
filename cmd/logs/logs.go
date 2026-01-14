package logs

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/kloudkit/ws-cli/internals/logger"
	"github.com/kloudkit/ws-cli/internals/styles"
)

var validLogLevels = []string{"debug", "info", "warn", "error"}

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

	if level != "" && !slices.Contains(validLogLevels, level) {
		styles.PrintErrorWithOptions(cmd.ErrOrStderr(), "Invalid log level. Must be one of:", [][]string{
			{"debug", "Debug information"},
			{"info", "General information"},
			{"warn", "Warning messages"},
			{"error", "Error messages only"},
		})
		return fmt.Errorf("invalid log level")
	}

	reader, err := logger.NewReader(tail, level)
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
}
