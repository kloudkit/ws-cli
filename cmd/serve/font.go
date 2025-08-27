package serve

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var fontCmd = &cobra.Command{
	Use:   "font",
	Short: "Serve fonts for local download",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		bind, _ := cmd.Flags().GetString("bind")

		host := strings.Join([]string{bind, ":", strconv.Itoa(port)}, "")

		handler := http.FileServer(http.Dir("/usr/share/fonts/"))

		fmt.Fprintln(cmd.OutOrStdout(), styles.SuccessStyle().Render(fmt.Sprintf("Serving fonts at %d", port)))
		fmt.Fprintln(cmd.OutOrStdout(), styles.InfoStyle().Render("To stop serving fonts, press Ctrl+C"))

		http.ListenAndServe(host, handler)
	},
}

func init() {
	ServeCmd.AddCommand(fontCmd)
}
