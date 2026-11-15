package mcp

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"gotest.tools/v3/assert"
)

type recorder struct {
	mu    sync.Mutex
	calls [][]string
}

func (r *recorder) run(_ context.Context, args []string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls = append(r.calls, args)

	return "ok", false
}

func (r *recorder) spawned() [][]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.calls)
}

func connect(t *testing.T, settings Settings, runner Runner) *sdk.ClientSession {
	t.Helper()
	useReference(t, referenceFixture)

	tools, _, err := BuildTools(syntheticTree(), settings)
	assert.NilError(t, err)

	serverSide, clientSide := sdk.NewInMemoryTransports()

	serverSession, err := NewServer("1.2.3", tools, runner).Connect(t.Context(), serverSide, nil)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	session, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).Connect(t.Context(), clientSide, nil)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	return session
}

func TestServerInMemory(t *testing.T) {
	t.Run("RejectedCallsSpawnNothing", func(t *testing.T) {
		spy := &recorder{}
		session := connect(t, Settings{}, spy.run)

		for arguments, want := range map[string]string{
			`{"follow":true}`:     "invalid arguments for logs: unknown argument [follow] (accepted: tail, level)",
			`{"tail":"5"}`:        "invalid arguments for logs: argument [tail] must be of type integer",
			`{"args":["--help"]}`: "invalid arguments for logs: unknown argument [args] (accepted: tail, level)",
		} {
			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "logs", Arguments: json.RawMessage(arguments)})
			assert.NilError(t, err)
			assert.Equal(t, result.Content[0].(*sdk.TextContent).Text, want)
			assert.Assert(t, result.IsError)
		}

		for _, name := range []string{"editor_open", "info_env", "editor_reap", "feature_store", "info"} {
			_, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: name, Arguments: json.RawMessage(`{}`)})
			assert.ErrorContains(t, err, "unknown tool", name)
		}

		assert.Equal(t, len(spy.spawned()), 0)

		_, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "logs", Arguments: json.RawMessage(`{"tail":5}`)})
		assert.NilError(t, err)
		assert.DeepEqual(t, spy.spawned(), [][]string{{"logs", "--tail=5"}})
	})

	t.Run("NoToolsStillAnswers", func(t *testing.T) {
		session := connect(t, Settings{Tools: []string{"bogus"}}, (&recorder{}).run)

		result, err := session.ListTools(t.Context(), nil)
		assert.NilError(t, err)
		assert.Equal(t, len(result.Tools), 0)
	})

	t.Run("Hints", func(t *testing.T) {
		read, write := describe(Tool{Name: "show_env", Access: Read}).Annotations, describe(Tool{Name: "editor_open", Access: Write}).Annotations

		assert.Assert(t, read.ReadOnlyHint && read.DestructiveHint == nil)
		assert.Assert(t, !write.ReadOnlyHint && write.DestructiveHint != nil && *write.DestructiveHint)
	})
}
