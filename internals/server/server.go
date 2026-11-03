package server

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/kloudkit/ws-cli/internals/logger"
	"github.com/kloudkit/ws-cli/internals/styles"
)

type Config struct {
	Port int
	Bind string
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.size += n
	return n, err
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

	return http.Serve(listener, accessLog(handler, w))
}

func accessLog(next http.Handler, w io.Writer) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: rw, status: http.StatusOK}

		next.ServeHTTP(rec, req)

		message := fmt.Sprintf("%s %s %d %s %s %d", req.Method, req.URL.Path, rec.status, time.Since(start), req.RemoteAddr, rec.size)
		logger.Log(w, "info", message, 0, true)
	})
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
