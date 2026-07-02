package show

import (
	"github.com/spf13/cobra"
)

var ShowCmd = &cobra.Command{
	Use:         "show",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Display information about the current workspace instance",
	Long:        "Resolve and print facts about this workspace instance — settings, IP addresses, and paths. --raw drops the styling for use in scripts.",
}

func init() {
	ShowCmd.PersistentFlags().Bool("raw", false, "Output raw value without styling")
}
