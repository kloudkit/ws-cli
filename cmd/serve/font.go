package serve

import (
	"github.com/kloudkit/ws-cli/internals/server"
	"github.com/spf13/cobra"
)

var fontCmd = &cobra.Command{
	Use:   "font",
	Short: "Serve fonts for local download",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		bind, _ := cmd.Flags().GetString("bind")

		config := server.Config{
			Port: port,
			Bind: bind,
		}

		return server.ServeDirectory(config, "/usr/share/fonts/", "fonts")
	},
}

func init() {
	ServeCmd.AddCommand(fontCmd)
}
