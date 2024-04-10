package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Interact with workspace related configurations",
}

var copyCmd = &cobra.Command{
	Use:   "cp [name]",
	Short: "Copying workspace defined configurations to a project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dest, _ := cmd.Flags().GetString("dest")

    dest, _ = filepath.Abs(dest)

    fmt.Println("Absolute path:", dest)
	},
}

func init() {
	copyCmd.Flags().String("dest", ".", "Output directory")
	copyCmd.Flags().BoolP("force", "f", false, "Force the overwriting of an existing file")

	ConfigCmd.AddCommand(copyCmd)
}
