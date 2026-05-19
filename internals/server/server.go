package server

import (
	"fmt"
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

func ServeDirectory(config Config, directory string, description string) error {
	host := formatAddr(config)

	handler := http.FileServer(http.Dir(directory))

	fmt.Println(styles.Success().Render(fmt.Sprintf("Serving %s at port %d", description, config.Port)))
	fmt.Println(styles.Info().Render("To stop serving, press Ctrl+C"))

	return http.ListenAndServe(host, handler)
}
