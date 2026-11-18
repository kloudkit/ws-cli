package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
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

func Listen(config Config) (net.Listener, error) {
	return net.Listen("tcp", net.JoinHostPort(config.Bind, strconv.Itoa(config.Port)))
}

func Serve(config Config, handler http.Handler, w io.Writer, announce func(url string)) error {
	listener, err := Listen(config)
	if err != nil {
		return err
	}

	return serve(listener, handler, w, announce)
}

func serve(listener net.Listener, handler http.Handler, w io.Writer, announce func(url string)) error {
	announce(DisplayURL(listener.Addr()))
	PrintStop(w)

	return Handle(context.Background(), listener, handler, w)
}

func Handle(ctx context.Context, listener net.Listener, handler http.Handler, w io.Writer) error {
	srv := &http.Server{Handler: accessLog(handler, w)}
	stop := context.AfterFunc(ctx, func() { _ = srv.Close() })
	defer stop()

	err := srv.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed) {
		return nil
	}

	return err
}

func Confined(dir string) (http.Handler, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}

	site := root.FS()
	files := http.FileServerFS(site)

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if !servable(site, req.URL.Path) {
			http.NotFound(rw, req)
			return
		}

		files.ServeHTTP(rw, req)
	}), nil
}

func servable(site fs.FS, urlPath string) bool {
	for _, segment := range strings.Split(urlPath, "/") {
		if strings.HasPrefix(segment, ".") {
			return false
		}
	}

	name := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	if name == "" {
		name = "."
	}

	info, err := fs.Stat(site, name)
	if err == nil && info.IsDir() {
		_, err = fs.Stat(site, path.Join(name, "index.html"))
	}

	return err == nil
}

func PrintStop(w io.Writer) {
	fmt.Fprintln(w)
	styles.PrintMuted(w, "Press Ctrl+C to stop")
}

func LogRequest(w io.Writer, fields ...any) {
	logger.Log(w, "info", strings.TrimSuffix(fmt.Sprintln(fields...), "\n"), 0, true)
}

func accessLog(next http.Handler, w io.Writer) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: rw, status: http.StatusOK}

		next.ServeHTTP(rec, req)

		LogRequest(w, req.Method, req.URL.Path, rec.status, time.Since(start), req.RemoteAddr, rec.size)
	})
}

func DisplayURL(addr net.Addr) string {
	host, port := "localhost", ""

	if tcp, ok := addr.(*net.TCPAddr); ok {
		port = strconv.Itoa(tcp.Port)
		if !tcp.IP.IsUnspecified() && !tcp.IP.IsLoopback() {
			host = tcp.IP.String()
		}
	}

	return "http://" + net.JoinHostPort(host, port)
}
