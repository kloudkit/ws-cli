package logs

import (
	"fmt"
	"os"

	"github.com/kloudkit/ws-cli/internals/logger"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var LogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Retrieve workspace logs",
	Args:  cobra.NoArgs,
	Run:   execute,
}

func execute(cmd *cobra.Command, args []string) {
	follow, _ := cmd.Flags().GetBool("follow")
	tail, _ := cmd.Flags().GetInt("tail")
	level, _ := cmd.Flags().GetString("level")

	if level != "" && level != "info" && level != "warn" && level != "error" && level != "debug" {
		fmt.Fprintln(cmd.ErrOrStderr(), styles.ErrorStyle().Render("Invalid log level. Must be one of: debug, info, warn, error"))
		os.Exit(1)
	}

	reader, err := logger.NewReader(tail, level)
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), styles.ErrorStyle().Render(fmt.Sprintf("Error: %s", err)))
		os.Exit(1)
	}

	if follow {
		err = reader.FollowLogs(cmd.OutOrStdout())
	} else {
		err = reader.ReadLogs(cmd.OutOrStdout())
	}

	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), styles.ErrorStyle().Render(fmt.Sprintf("Error reading logs: %s", err)))
		os.Exit(1)
	}
}

func init() {
	LogsCmd.Flags().BoolP("follow", "f", false, "Follow log output in real-time")
	LogsCmd.Flags().IntP("tail", "t", 0, "Number of lines to show from the end (0 for all)")
	LogsCmd.Flags().StringP("level", "l", "", "Filter by log level (debug|info|warn|error)")
}
