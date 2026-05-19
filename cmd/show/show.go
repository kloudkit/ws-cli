package show

import (
	"github.com/spf13/cobra"
)

var ShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display information about the current workspace instance",
}

func init() {
	ShowCmd.PersistentFlags().Bool("raw", false, "Output raw value without styling")
}
