package serve

import (
	"os"

	"github.com/kloudkit/ws-cli/internals/server"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Serve current directory as a static site",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		bind, _ := cmd.Flags().GetString("bind")

		config := server.Config{
			Port: port,
			Bind: bind,
		}

		currentDir, err := os.Getwd()
		if err != nil {
			cmd.PrintErrf("Error getting current directory: %v\n", err)
			os.Exit(1)
		}

		if err := server.ServeDirectory(config, currentDir, "current directory"); err != nil {
			cmd.PrintErrf("Error starting server: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	ServeCmd.AddCommand(currentCmd)
}
