package server

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kloudkit/ws-cli/internals/logger"
)

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

func accessLogMiddleware(next http.Handler, w io.Writer) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: rw, status: http.StatusOK}

		next.ServeHTTP(rec, req)

		logger.Log(
			w,
			"info",
			fmt.Sprintf(
				"%s %s %d %s %s %d",
				req.Method,
				req.URL.Path,
				rec.status,
				time.Since(start),
				req.RemoteAddr,
				rec.size,
			),
			0,
			true,
		)
	})
}
