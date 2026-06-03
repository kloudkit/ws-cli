package server

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"

	"github.com/kloudkit/ws-cli/internals/styles"
)

type Config struct {
	Port int
	Bind string
}

func formatAddr(c Config) string {
	return net.JoinHostPort(c.Bind, strconv.Itoa(c.Port))
}

func Serve(config Config, handler http.Handler, description string, w io.Writer) error {
	host := formatAddr(config)

	fmt.Fprintln(w, styles.Success().Render(fmt.Sprintf("Serving %s at port %d", description, config.Port)))
	fmt.Fprintln(w, styles.Info().Render("To stop serving, press Ctrl+C"))

	return http.ListenAndServe(host, accessLogMiddleware(handler, w))
}

func ServeDirectory(config Config, directory string, description string, w io.Writer) error {
	return Serve(config, http.FileServer(http.Dir(directory)), description, w)
}
