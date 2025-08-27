package serve

import (
	"github.com/spf13/cobra"
	"github.com/kloudkit/ws-cli/cmd/serve/font"
)

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve internal assets",
}

func init() {
	ServeCmd.PersistentFlags().IntP("port", "p", 38080, "Port to serve assets on")
	ServeCmd.PersistentFlags().String("bind", "0.0.0.0", "Bind address")
	ServeCmd.AddCommand(font.FontCmd)
}
