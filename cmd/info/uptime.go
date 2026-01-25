package info

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/styles"
)

func humanizeDuration(duration time.Duration) string {
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	var parts []string

	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d days", days))
	}

	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d hours", hours))
	}

	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d minutes", minutes))
	}

	if len(parts) == 0 {
		return "just now"
	}

	return strings.Join(parts, ", ") + " ago"
}

var uptimeCmd = &cobra.Command{
	Use:   "uptime",
	Short: "Display the workspace uptime",
	RunE: func(cmd *cobra.Command, args []string) error {
		started, running, err := config.GetSessionInfo()

		if err != nil {
			styles.PrintWarning(cmd.OutOrStdout(), "Could not read workspace startup time")
			return nil
		}

		var statusValue string
		switch {
		case running.Hours() < 1:
			statusValue = "Recently started"
		case running.Hours() < 36:
			statusValue = "Active"
		default:
			statusValue = "Long running"
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Title().Render("Uptime"))

		t := styles.Table().Rows(
			[]string{"Started at", styles.Code().Render(started.Format("2006-01-02 15:04:05 MST"))},
			[]string{"Running for", humanizeDuration(running)},
			[]string{"Status", statusValue},
		)

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", t.Render())
		return nil
	},
}

func init() {
	InfoCmd.AddCommand(uptimeCmd)
}
