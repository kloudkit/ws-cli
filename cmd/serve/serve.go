package serve

import "github.com/spf13/cobra"

var ServeCmd = &cobra.Command{
	Use:         "serve",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Serve internal assets",
	Long:        "Run a small HTTP server for local assets — fonts or the current directory — on --port (default 38080).",
	Example: `# Serve the current directory
ws serve current

# Serve fonts on a custom port
ws serve font --port 38081`,
}

func init() {
	ServeCmd.PersistentFlags().IntP("port", "p", 38080, "Port to serve assets on")
	ServeCmd.PersistentFlags().String("bind", "0.0.0.0", "Bind address")

	ServeCmd.AddCommand(fontCmd, currentCmd)
}
