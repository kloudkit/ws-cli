package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func stripAnsi(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(s, "")
}

func TestResponseRecorderStatusAndSize(t *testing.T) {
	cases := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantSize   int
	}{
		{
			"defaults to 200 when WriteHeader not called",
			func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("hello")) },
			http.StatusOK,
			5,
		},
		{
			"captures explicit 404 with empty body",
			func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) },
			http.StatusNotFound,
			0,
		},
		{
			"captures explicit 500 with empty body",
			func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			http.StatusInternalServerError,
			0,
		},
		{
			"captures size after explicit 200",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("twelve bytes"))
			},
			http.StatusOK,
			12,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := &responseRecorder{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}

			c.handler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

			assert.Equal(t, c.wantStatus, rec.status)
			assert.Equal(t, c.wantSize, rec.size)
		})
	}
}

func TestAccessLogMiddlewareLineFormat(t *testing.T) {
	buffer := new(bytes.Buffer)

	handler := accessLogMiddleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}),
		buffer,
	)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.RemoteAddr = "10.0.0.5:53124"

	handler.ServeHTTP(httptest.NewRecorder(), req)

	line := strings.TrimRight(stripAnsi(buffer.String()), "\n")

	assert.Assert(t, cmp.Regexp(`^\[.*?\]\s+(\w+)\s*(.*)$`, line))
	assert.Assert(t, strings.Contains(line, "info"))
	assert.Assert(t, strings.Contains(line, "GET"))
	assert.Assert(t, strings.Contains(line, "/healthz"))
	assert.Assert(t, strings.Contains(line, "200"))
	assert.Assert(t, strings.Contains(line, "10.0.0.5:53124"))
	assert.Assert(t, cmp.Regexp(`\d+(\.\d+)?(ns|µs|ms|s)`, line))
}

func TestAccessLogServesAndLogs(t *testing.T) {
	buffer := new(bytes.Buffer)
	rec := httptest.NewRecorder()

	handler := accessLogMiddleware(http.FileServer(http.Dir("/tmp")), buffer)
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Assert(t, rec.Body.Len() > 0)
	assert.Assert(t, strings.Contains(stripAnsi(buffer.String()), "GET /"))
}
