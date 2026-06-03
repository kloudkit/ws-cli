package logs

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

func executeLogs(args ...string) (string, error) {
	buffer := new(bytes.Buffer)

	LogsCmd.Flags().Set("target", "main")
	LogsCmd.Flags().Set("tail", "0")
	LogsCmd.Flags().Set("level", "")
	LogsCmd.Flags().Set("follow", "false")

	cmd := &cobra.Command{}
	cmd.AddCommand(LogsCmd)
	cmd.SetOut(buffer)
	cmd.SetErr(buffer)
	cmd.SetArgs(append([]string{"logs"}, args...))

	err := cmd.Execute()

	return buffer.String(), err
}

func _seedLogs(t *testing.T) {
	t.Helper()

	tempDir := t.TempDir()

	files := map[string]string{
		"workspace.log":  "main marker",
		"metrics.log":    "metrics marker",
		"dockerd.log":    "docker marker",
		"auth-proxy.log": "auth proxy marker",
	}

	for name, content := range files {
		assert.NilError(t, os.WriteFile(filepath.Join(tempDir, name), []byte(content+"\n"), 0644))
	}

	t.Setenv("WS_LOGGING_DIR", tempDir)
	t.Setenv("WS_LOGGING_MAIN_FILE", "workspace.log")
	t.Setenv("WS_LOGGING_METRICS_FILE", "metrics.log")
	t.Setenv("WS_LOGGING_DOCKER_FILE", "dockerd.log")
	t.Setenv("WS_LOGGING_AUTH_PROXY_FILE", "auth-proxy.log")
}

func TestLogsDefaultTargetIsMain(t *testing.T) {
	assert.Equal(t, "main", LogsCmd.Flags().Lookup("target").DefValue)
}

func TestLogsNoShortTargetAlias(t *testing.T) {
	assert.Assert(t, LogsCmd.Flags().ShorthandLookup("T") == nil)
}

func TestLogsValidTargets(t *testing.T) {
	tests := []struct {
		target string
		marker string
	}{
		{"main", "main marker"},
		{"metrics", "metrics marker"},
		{"docker", "docker marker"},
		{"auth_proxy", "auth proxy marker"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			_seedLogs(t)

			out, err := executeLogs("--target=" + tt.target)
			assert.NilError(t, err)
			assert.Assert(t, strings.Contains(out, tt.marker))
		})
	}
}

func TestLogsInvalidTargetRejected(t *testing.T) {
	out, err := executeLogs("--target=garbage")

	assert.ErrorContains(t, err, "invalid log target")
	assert.Assert(t, strings.Contains(out, "Invalid log target"))
}

func TestLogsTargetComposesWithTailAndLevel(t *testing.T) {
	_seedLogs(t)

	out, err := executeLogs("--target=metrics", "--tail=5", "--level=info")

	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(out, "metrics marker"))
}
