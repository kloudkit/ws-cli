package info

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"os"
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

var uptimeCmd = &cobra.Command{
	Use:   "uptime",
	Short: "Display the workspace uptime",
	Run: func(cmd *cobra.Command, args []string) {
		started, running, _ := readStartup()

		fmt.Fprintln(cmd.OutOrStdout(), "Uptime")
		fmt.Fprintln(cmd.OutOrStdout(), "  started\t", started)
		fmt.Fprintln(cmd.OutOrStdout(), "  running\t", running)
		fmt.Fprintln(cmd.OutOrStdout(), "  human-readable\t", "@todo:")
	},
}

func init() {
	InfoCmd.AddCommand(uptimeCmd)
}
