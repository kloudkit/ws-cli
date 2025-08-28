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
	RunE:  execute,
}

func execute(cmd *cobra.Command, args []string) error {
	follow, _ := cmd.Flags().GetBool("follow")
	tail, _ := cmd.Flags().GetInt("tail")
	level, _ := cmd.Flags().GetString("level")

	if level != "" && level != "info" && level != "warn" && level != "error" && level != "debug" {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n\n", styles.ErrorBadge().Render("ERROR"))
		fmt.Fprintln(cmd.ErrOrStderr(), styles.Error().Render("Invalid log level. Must be one of:"))
		fmt.Fprintf(cmd.ErrOrStderr(), "  %s %s\n", styles.Code().Render("debug"), styles.Muted().Render("- Debug information"))
		fmt.Fprintf(cmd.ErrOrStderr(), "  %s %s\n", styles.Code().Render("info"), styles.Muted().Render("- General information"))
		fmt.Fprintf(cmd.ErrOrStderr(), "  %s %s\n", styles.Code().Render("warn"), styles.Muted().Render("- Warning messages"))
		fmt.Fprintf(cmd.ErrOrStderr(), "  %s %s\n", styles.Code().Render("error"), styles.Muted().Render("- Error messages only"))
		os.Exit(1)
	}

	reader, err := logger.NewReader(tail, level)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n\n", styles.ErrorBadge().Render("ERROR"))
		fmt.Fprintln(cmd.ErrOrStderr(), styles.Error().Render(fmt.Sprintf("Failed to initialize log reader: %s", err)))
		os.Exit(1)
	}

	if follow {
		err = reader.FollowLogs(cmd.OutOrStdout())
	} else {
		err = reader.ReadLogs(cmd.OutOrStdout())
	}

	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n\n", styles.ErrorBadge().Render("ERROR"))
		fmt.Fprintln(cmd.ErrOrStderr(), styles.Error().Render(fmt.Sprintf("Error reading logs: %s", err)))
		os.Exit(1)
	}

	return nil
}

func init() {
	LogsCmd.Flags().BoolP("follow", "f", false, "Follow log output in real-time")
	LogsCmd.Flags().IntP("tail", "t", 0, "Number of lines to show from the end (0 for all)")
	LogsCmd.Flags().StringP("level", "l", "", "Filter by log level (debug|info|warn|error)")
}
