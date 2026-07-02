package clip

import (
	"github.com/spf13/cobra"
)

var ClipCmd = &cobra.Command{
	Use:         "clip",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Interact with the native clipboard",
	Long:        "Reach the browser clipboard from the terminal over the workspace IPC socket.",
}
