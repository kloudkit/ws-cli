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

func Serve(config Config, handler http.Handler, description string, w io.Writer) error {
	fmt.Fprintln(w, styles.Success().Render(fmt.Sprintf("Serving %s at port %d", description, config.Port)))
	fmt.Fprintln(w, styles.Info().Render("To stop serving, press Ctrl+C"))

	host := net.JoinHostPort(config.Bind, strconv.Itoa(config.Port))
	return http.ListenAndServe(host, accessLogMiddleware(handler, w))
}
