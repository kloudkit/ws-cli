package clip

import (
	"github.com/spf13/cobra"
)

var ClipCmd = &cobra.Command{
	Use:   "clip",
	Short: "Interact with the native clipboard",
}
