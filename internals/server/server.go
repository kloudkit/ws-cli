package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kloudkit/ws-cli/internals/styles"
)

type Config struct {
	Port int
	Bind string
}

func ServeDirectory(config Config, directory string, description string) error {
	host := strings.Join([]string{config.Bind, ":", strconv.Itoa(config.Port)}, "")

	handler := http.FileServer(http.Dir(directory))

	fmt.Println(styles.Success().Render(fmt.Sprintf("Serving %s at port %d", description, config.Port)))
	fmt.Println(styles.Info().Render("To stop serving, press Ctrl+C"))

	return http.ListenAndServe(host, handler)
}
