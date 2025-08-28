package serve

import (
	"fmt"
	"os"

	"github.com/kloudkit/ws-cli/internals/server"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Serve current directory as a static site",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		bind, _ := cmd.Flags().GetString("bind")

		config := server.Config{
			Port: port,
			Bind: bind,
		}

		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current directory: %v", err)
		}

		return server.ServeDirectory(config, currentDir, "current directory")
	},
}

func init() {
	ServeCmd.AddCommand(currentCmd)
}
