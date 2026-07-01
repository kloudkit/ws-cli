package clip

import (
	"github.com/spf13/cobra"
)

var ClipCmd = &cobra.Command{
	Use:   "clip",
	Short: "Interact with the native clipboard",
	Long:  "Reach the browser clipboard from the terminal over the workspace IPC socket.",
}
