package info

import (
	"fmt"
	"io"

	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
	"os"
	"strings"
	"time"
)

func readStartup() (time.Time, time.Duration, error) {
	data, err := os.ReadFile("/var/lib/workspace/state/initialized")
	if err != nil {
		return time.Time{}, 0, err
	}

	parsedTime, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		return time.Time{}, 0, err
	}

	return parsedTime, time.Since(parsedTime), nil
}

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

func showUptime(writer io.Writer) {
	started, running, err := readStartup()

	if err != nil {
		fmt.Fprintf(writer, "%s\n", styles.Title().Render("Warning"))
		fmt.Fprintf(writer, "%s\n", styles.Warning().Render("Could not read workspace startup time"))
		return
	}

	var statusValue string
	if running.Hours() < 1 {
		statusValue = "Recently started"
	} else if running.Hours() < 36 {
		statusValue = "Active"
	} else {
		statusValue = "Long running"
	}

	fmt.Fprintf(writer, "%s\n", styles.Title().Render("Uptime"))

	t := styles.Table().Rows(
		[]string{"Started at", styles.Code().Render(started.Format("2006-01-02 15:04:05 MST"))},
		[]string{"Running for", styles.Value().Render(humanizeDuration(running))},
		[]string{"Status", styles.Value().Render(statusValue)},
	)

	fmt.Fprintf(writer, "%s\n\n", t.Render())
}

var uptimeCmd = &cobra.Command{
	Use:   "uptime",
	Short: "Display the workspace uptime",
	RunE: func(cmd *cobra.Command, args []string) error {
		showUptime(cmd.OutOrStdout())
		return nil
	},
}

func init() {
	InfoCmd.AddCommand(uptimeCmd)
}
