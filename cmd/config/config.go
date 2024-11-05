package config

import (
	"github.com/kloudkit/ws-cli/cmd/config/copy"
	"github.com/spf13/cobra"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Interact with workspace related configurations",
}

func init() {
	ConfigCmd.AddCommand(copy.CopyCmd)
}
