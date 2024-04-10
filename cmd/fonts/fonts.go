package fonts

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var FontsCmd = &cobra.Command{
	Use:   "fonts",
	Short: "Font related assets",
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve fonts for local download",
	Run: func(cmd *cobra.Command, args []string) {
    port, _ := cmd.Flags().GetInt("port")

    host := strings.Join([]string{"0.0.0.0", ":", strconv.Itoa(port)}, "")

    handler := http.FileServer(http.Dir("/usr/share/fonts/"))

    fmt.Fprintln(cmd.OutOrStdout(), "Serving fonts at", port)
    fmt.Fprintln(cmd.OutOrStdout(), "To stop serving fonts, press Ctrl+C")

		http.ListenAndServe(host, handler)
	},
}

func init() {
  serveCmd.Flags().IntP("port", "p", 38080, "Port to serve fonts on")

	FontsCmd.AddCommand(serveCmd)
}
