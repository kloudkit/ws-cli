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
	started, running, _ := readStartup()

	fmt.Fprintln(writer, styles.Header().Render("Uptime"))
	fmt.Fprintln(writer)

	t := styles.Table("", "Value").
		Rows(
			[]string{"started", started.String()},
			[]string{"running", humanizeDuration(running)},
		)

	fmt.Fprintln(writer, t.Render())
}

var uptimeCmd = &cobra.Command{
	Use:   "uptime",
	Short: "Display the workspace uptime",
	Run: func(cmd *cobra.Command, args []string) {
		showUptime(cmd.OutOrStdout())
	},
}

func init() {
	InfoCmd.AddCommand(uptimeCmd)
}
