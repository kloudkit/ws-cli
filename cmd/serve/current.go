package serve

import (
	"fmt"
	"net/http"
	"os"

	"github.com/kloudkit/ws-cli/internals/server"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:         "current",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Serve current directory as a static site",
	Long:        "Serve the current directory over HTTP as a static site — a quick way to preview built files.",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		bind, _ := cmd.Flags().GetString("bind")

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Title().Render("Static server"))

		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current directory: %v", err)
		}

		handler := http.FileServer(http.Dir(currentDir))

		return server.Serve(server.Config{Port: port, Bind: bind}, handler, "current directory", cmd.OutOrStdout())
	},
}

func init() {
	ServeCmd.AddCommand(currentCmd)
}
