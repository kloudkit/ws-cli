package mcp

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"gotest.tools/v3/assert"
)

type recorder struct {
	mu     sync.Mutex
	calls  [][]string
	text   string
	failed bool
}

func (r *recorder) run(_ context.Context, args []string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls = append(r.calls, args)

	return r.text, r.failed
}

func (r *recorder) spawned() [][]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.calls)
}

func connect(t *testing.T, settings Settings, runner Runner, protocol string) *sdk.ClientSession {
	t.Helper()
	useReference(t, referenceFixture)

	tools, _, err := BuildTools(syntheticTree(), settings)
	assert.NilError(t, err)

	serverSide, clientSide := sdk.NewInMemoryTransports()

	serverSession, err := NewServer("1.2.3", tools, runner).Connect(t.Context(), serverSide, nil)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(t.Context(), clientSide, &sdk.ClientSessionOptions{ProtocolVersion: protocol})
	assert.NilError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	return session
}

func listed(t *testing.T, session *sdk.ClientSession) map[string]*sdk.Tool {
	t.Helper()

	result, err := session.ListTools(t.Context(), nil)
	assert.NilError(t, err)

	tools := map[string]*sdk.Tool{}
	for _, tool := range result.Tools {
		tools[tool.Name] = tool
	}

	return tools
}

func call(t *testing.T, session *sdk.ClientSession, name, arguments string) (string, bool) {
	t.Helper()

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: name, Arguments: json.RawMessage(arguments)})
	assert.NilError(t, err)
	assert.Equal(t, len(result.Content), 1)

	return result.Content[0].(*sdk.TextContent).Text, result.IsError
}

func sortedNames(tools map[string]*sdk.Tool) string {
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}

	slices.Sort(names)

	return strings.Join(names, " ")
}

func TestServerInMemory(t *testing.T) {
	const latest = "2026-07-28"

	t.Run("DefaultListsReadToolsOnly", func(t *testing.T) {
		spy := &recorder{}
		session := connect(t, Settings{}, spy.run, latest)

		tools := listed(t, session)
		assert.Equal(t, sortedNames(tools), "editor_list info_version logs show_env show_path_vscode_settings")

		for name, tool := range tools {
			assert.Assert(t, tool.Annotations.ReadOnlyHint, name)
			assert.Assert(t, tool.Annotations.DestructiveHint == nil, name)
		}

		for _, name := range []string{"editor_open", "info_env", "editor_reap", "feature_store", "info", "serve_mcp"} {
			_, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: name, Arguments: json.RawMessage(`{}`)})
			assert.ErrorContains(t, err, "unknown tool", name)
		}

		assert.Equal(t, len(spy.spawned()), 0)
	})

	t.Run("WritesEnabled", func(t *testing.T) {
		spy := &recorder{text: "opened"}
		session := connect(t, Settings{AllowWrite: true}, spy.run, latest)

		tools := listed(t, session)
		assert.Equal(t, sortedNames(tools), "editor_list editor_open info_version logs show_env show_path_vscode_settings")

		open := tools["editor_open"]
		assert.Assert(t, !open.Annotations.ReadOnlyHint)
		assert.Assert(t, open.Annotations.DestructiveHint != nil && *open.Annotations.DestructiveHint)
		assert.Equal(t, open.Title, "Short open <file>")
		assert.Equal(t, open.Description, "Open a file in the editor.")

		text, failed := call(t, session, "editor_open", `{"args":["main.go"]}`)
		assert.Equal(t, text, "opened")
		assert.Assert(t, !failed)
		assert.DeepEqual(t, spy.spawned(), [][]string{{"editor", "open", "--", "main.go"}})
	})

	t.Run("SchemaReachesTheClient", func(t *testing.T) {
		session := connect(t, Settings{}, (&recorder{}).run, latest)

		schema, err := json.Marshal(listed(t, session)["logs"].InputSchema)
		assert.NilError(t, err)
		assert.Equal(t, string(schema),
			`{"additionalProperties":false,"properties":{"level":{"description":"Level filter","type":"string"},"tail":{"description":"Lines from the end","type":"integer"}},"type":"object"}`)
	})

	t.Run("InvalidArgumentsSpawnNothing", func(t *testing.T) {
		spy := &recorder{}
		session := connect(t, Settings{}, spy.run, latest)

		for arguments, want := range map[string]string{
			`{"follow":true}`:     "invalid arguments for logs: unknown argument [follow] (accepted: tail, level)",
			`{"tail":"5"}`:        "invalid arguments for logs: argument [tail] must be of type integer",
			`{"args":["--help"]}`: "invalid arguments for logs: unknown argument [args] (accepted: tail, level)",
		} {
			text, failed := call(t, session, "logs", arguments)
			assert.Equal(t, text, want)
			assert.Assert(t, failed)
		}

		assert.Equal(t, len(spy.spawned()), 0)
	})

	t.Run("BuildsArgv", func(t *testing.T) {
		spy := &recorder{text: "ok"}
		session := connect(t, Settings{}, spy.run, latest)

		call(t, session, "logs", `{"level":"error","tail":5}`)
		call(t, session, "show_env", `{"args":["--value"]}`)
		call(t, session, "show_path_vscode_settings", `{}`)
		call(t, session, "info_version", `{}`)

		assert.DeepEqual(t, spy.spawned(), [][]string{
			{"logs", "--tail=5", "--level=error"},
			{"show", "env", "--", "--value"},
			{"show", "path", "vscode-settings"},
			{"info", "version"},
		})
	})

	t.Run("FailureIsAToolError", func(t *testing.T) {
		session := connect(t, Settings{}, (&recorder{text: "exit code 1", failed: true}).run, latest)

		text, failed := call(t, session, "info_version", `{}`)
		assert.Equal(t, text, "exit code 1")
		assert.Assert(t, failed)
	})

	t.Run("NoToolsStillAnswers", func(t *testing.T) {
		session := connect(t, Settings{Tools: []string{"bogus"}}, (&recorder{}).run, latest)

		assert.Equal(t, len(listed(t, session)), 0)
	})

	for _, protocol := range []string{"2025-11-25", latest, "2025-06-18"} {
		t.Run("Protocol"+protocol, func(t *testing.T) {
			spy := &recorder{text: "1.2.3"}
			session := connect(t, Settings{}, spy.run, protocol)

			assert.Equal(t, session.InitializeResult().ProtocolVersion, protocol)
			assert.DeepEqual(t, session.InitializeResult().ServerInfo, &sdk.Implementation{Name: "ws-cli", Version: "1.2.3"})
			assert.Equal(t, sortedNames(listed(t, session)), "editor_list info_version logs show_env show_path_vscode_settings")

			text, failed := call(t, session, "info_version", `{}`)
			assert.Equal(t, text, "1.2.3")
			assert.Assert(t, !failed)
		})
	}

	t.Run("RunsTheRealRunner", func(t *testing.T) {
		useHelper(t)
		session := connect(t, Settings{}, Run, latest)

		text, failed := call(t, session, "show_env", `{"args":["--value"]}`)
		assert.Equal(t, text, "show|env|--|--value")
		assert.Assert(t, !failed)
	})
}
