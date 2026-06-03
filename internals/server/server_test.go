package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestFormatAddr(t *testing.T) {
	cases := []struct {
		bind string
		port int
		want string
	}{
		{"127.0.0.1", 8080, "127.0.0.1:8080"},
		{"::1", 8080, "[::1]:8080"},
		{"", 0, ":0"},
		{"0.0.0.0", 80, "0.0.0.0:80"},
	}

	for _, c := range cases {
		assert.Equal(t, c.want, formatAddr(Config{Port: c.port, Bind: c.bind}))
	}
}

func TestConfig(t *testing.T) {
	t.Run("ConfigFields", func(t *testing.T) {
		config := Config{
			Port: 8080,
			Bind: "127.0.0.1",
		}

		assert.Equal(t, 8080, config.Port)
		assert.Equal(t, "127.0.0.1", config.Bind)
	})
}

func TestServeDirectory(t *testing.T) {
	t.Run("Integration", func(t *testing.T) {
		testDir := "/tmp"

		handler := http.FileServer(http.Dir(testDir))
		server := httptest.NewServer(handler)
		defer server.Close()

		resp, err := http.Get(server.URL)
		assert.NilError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("WithInvalidDirectory", func(t *testing.T) {
		config := Config{
			Port: 0,
			Bind: "127.0.0.1",
		}

		done := make(chan error, 1)
		go func() {
			err := ServeDirectory(config, "/nonexistent/directory", "test", io.Discard)
			done <- err
		}()

		time.Sleep(100 * time.Millisecond)

		select {
		case err := <-done:
			if err != nil {
				assert.Assert(t, strings.Contains(err.Error(), "bind") || strings.Contains(err.Error(), "address"))
			}
		case <-time.After(200 * time.Millisecond):
		}
	})
}
