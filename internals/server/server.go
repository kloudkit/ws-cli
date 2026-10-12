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

func Serve(config Config, handler http.Handler, w io.Writer, announce func(url string)) error {
	listener, err := net.Listen("tcp", net.JoinHostPort(config.Bind, strconv.Itoa(config.Port)))
	if err != nil {
		return err
	}

	return serve(listener, handler, w, announce)
}

func serve(listener net.Listener, handler http.Handler, w io.Writer, announce func(url string)) error {
	announce(displayURL(listener.Addr()))

	fmt.Fprintln(w)
	styles.PrintMuted(w, "Press Ctrl+C to stop")

	return http.Serve(listener, accessLogMiddleware(handler, w))
}

func displayURL(addr net.Addr) string {
	host, port := "localhost", ""

	if tcp, ok := addr.(*net.TCPAddr); ok {
		port = strconv.Itoa(tcp.Port)
		if !tcp.IP.IsUnspecified() && !tcp.IP.IsLoopback() {
			host = tcp.IP.String()
		}
	}

	return "http://" + net.JoinHostPort(host, port)
}
