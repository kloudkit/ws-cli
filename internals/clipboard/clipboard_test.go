package clipboard_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/clipboard"
	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
)

func TestPasteWritesDecodedString(t *testing.T) {
	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var envelope map[string]any
		_ = json.Unmarshal(body, &envelope)
		assert.Equal(t, envelope["type"], "clipboardRead")

		_, _ = w.Write([]byte(`"line one\nline two"`))
	}))

	var out bytes.Buffer
	assert.NilError(t, clipboard.Paste(&out))
	assert.Equal(t, out.String(), "line one\nline two")
}

func TestPasteNullIsEmpty(t *testing.T) {
	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("null"))
	}))

	var out bytes.Buffer
	assert.NilError(t, clipboard.Paste(&out))
	assert.Equal(t, out.String(), "")
}

func TestCopySendsStdinAsContent(t *testing.T) {
	var envelope map[string]any
	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &envelope)
	}))

	assert.NilError(t, clipboard.Copy(strings.NewReader("line one\nline two")))
	assert.Equal(t, envelope["type"], "clipboard")
	assert.Equal(t, envelope["content"], "line one\nline two")
}
