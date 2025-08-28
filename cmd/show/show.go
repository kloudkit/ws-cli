package show

import (
	"github.com/spf13/cobra"
)

var ShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show information about the current workspace instance",
}

func init() {
	ShowCmd.AddCommand(pathCmd, ipCmd)
}
