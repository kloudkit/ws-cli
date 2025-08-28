package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kloudkit/ws-cli/internals/styles"
)

// Config holds the server configuration
type Config struct {
	Port int
	Bind string
}

// ServeDirectory serves a directory with HTTP file server
func ServeDirectory(config Config, directory string, description string) error {
	host := strings.Join([]string{config.Bind, ":", strconv.Itoa(config.Port)}, "")

	handler := http.FileServer(http.Dir(directory))

	fmt.Println(styles.SuccessStyle().Render(fmt.Sprintf("Serving %s at port %d", description, config.Port)))
	fmt.Println(styles.InfoStyle().Render("To stop serving, press Ctrl+C"))

	return http.ListenAndServe(host, handler)
}
