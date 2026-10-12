package clip

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/testutil"
	"gotest.tools/v3/assert"
)

func TestPasteWritesClipboard(t *testing.T) {
	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`"copied text\n"`))
	}))

	var out bytes.Buffer
	ClipCmd.SetOut(&out)
	ClipCmd.SetArgs([]string{"paste"})

	assert.NilError(t, ClipCmd.Execute())
	assert.Equal(t, out.String(), "copied text\n")
}

func TestCopySendsStdin(t *testing.T) {
	var envelope map[string]any
	testutil.StartIPC(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &envelope)
	}))

	ClipCmd.SetIn(strings.NewReader("from stdin"))
	ClipCmd.SetArgs([]string{"copy"})

	assert.NilError(t, ClipCmd.Execute())
	assert.Equal(t, envelope["type"], "clipboard")
	assert.Equal(t, envelope["content"], "from stdin")
}

func TestPasteWithoutSocketFails(t *testing.T) {
	t.Setenv("VSCODE_IPC_HOOK_CLI", filepath.Join(t.TempDir(), "absent.sock"))

	var out bytes.Buffer
	ClipCmd.SetOut(&out)
	ClipCmd.SetErr(&out)
	ClipCmd.SetArgs([]string{"paste"})

	assert.Assert(t, ClipCmd.Execute() != nil)
	assert.Assert(t, !strings.Contains(out.String(), "\x1b"))
}
