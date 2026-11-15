package cmd_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/kloudkit/ws-cli/cmd"
	"github.com/kloudkit/ws-cli/internals/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

const mcpEnvFixture = `
envs:
  server:
    properties:
      root:
        type: string
        default: /workspace
        description: Root directory for the workspace.
      port:
        type: integer
        default: 8080
        description: Port on which the web server listens.
  auth:
    properties:
      password:
        type: string
        default: null
        secret: true
        description: Password for the editor.
      token:
        type: string
        default: null
        secret: true
        description: Token for the editor.
      key:
        type: string
        default: null
        secret: true
        description: Key for the editor.
  features:
    properties:
      store_url:
        type: string
        default: null
  ai:
    properties:
      mcp_allow_write:
        type: boolean
        default: false
      mcp_tools:
        type: string
        default: "*"
        delimiter: ","
`

type pinnedTool struct {
	Access     string
	When       string
	Flags      string
	Properties string
}

var pinnedTools = map[string]pinnedTool{
	"editor_diagnostics":        {Access: "read", Flags: "uri", Properties: "uri"},
	"editor_list":               {Access: "read"},
	"editor_open":               {Access: "write", Flags: "selection,preview", Properties: "args,preview,selection"},
	"editor_reveal":             {Access: "write", Properties: "args"},
	"editor_selection":          {Access: "read"},
	"feature_info":              {Access: "read", Properties: "args"},
	"feature_list":              {Access: "read"},
	"feature_store":             {Access: "read", When: "features.store_url"},
	"info_extensions":           {Access: "read"},
	"info_metrics":              {Access: "read", Flags: "gpu", Properties: "gpu"},
	"info_uptime":               {Access: "read"},
	"info_version":              {Access: "read", Flags: "all", Properties: "all"},
	"logs":                      {Access: "read", Flags: "tail,level,target", Properties: "level,tail,target"},
	"seed_ls":                   {Access: "read"},
	"show_env":                  {Access: "read", Properties: "args"},
	"show_ip_internal":          {Access: "read"},
	"show_ip_node":              {Access: "read"},
	"show_path_home":            {Access: "read"},
	"show_path_vscode_settings": {Access: "read", Flags: "workspace", Properties: "workspace"},
	"template_apply":            {Access: "write", Flags: "path", Properties: "args,path"},
	"template_list":             {Access: "read"},
	"template_show":             {Access: "read", Flags: "local", Properties: "args,local"},
}

func (w *workspace) withReference(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(w.dir, "mcp.reference.yaml")
	assert.NilError(t, os.WriteFile(path, []byte(content), 0o644))

	return "WS__INTERNAL_ENV_REFERENCE=" + path
}

func (w *workspace) server(extraEnv ...string) *exec.Cmd {
	command := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	command.Dir = filepath.Join(w.dir, "home")
	command.Env = append(append([]string{helperArgsEnv + "=serve\x1fmcp"}, w.env...), extraEnv...)

	return command
}

func (w *workspace) connect(t *testing.T, protocol string, extraEnv ...string) *sdk.ClientSession {
	t.Helper()

	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(t.Context(), &sdk.CommandTransport{Command: w.server(extraEnv...)}, &sdk.ClientSessionOptions{ProtocolVersion: protocol})
	assert.NilError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	return session
}

func listTools(t *testing.T, session *sdk.ClientSession) map[string]*sdk.Tool {
	t.Helper()

	result, err := session.ListTools(t.Context(), nil)
	assert.NilError(t, err)

	tools := map[string]*sdk.Tool{}
	for _, tool := range result.Tools {
		tools[tool.Name] = tool
	}

	return tools
}

func callTool(t *testing.T, session *sdk.ClientSession, name string, arguments any) (string, bool) {
	t.Helper()

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: name, Arguments: arguments})
	assert.NilError(t, err, name)
	assert.Equal(t, len(result.Content), 1, name)

	return result.Content[0].(*sdk.TextContent).Text, result.IsError
}

func properties(t *testing.T, tool *sdk.Tool) []string {
	t.Helper()

	schema, ok := tool.InputSchema.(map[string]any)
	assert.Assert(t, ok, tool.Name)
	assert.Equal(t, schema["type"], "object", tool.Name)
	assert.Equal(t, schema["additionalProperties"], false, tool.Name)

	declared, _ := schema["properties"].(map[string]any)

	return slices.Sorted(maps.Keys(declared))
}

func TestMcpToolSurfaceIsPinned(t *testing.T) {
	annotated := map[string]pinnedTool{}

	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if spec, ok, err := mcp.ParseSpec(c.Annotations); err == nil && ok {
			terms := make([]string, len(spec.When))
			for i, term := range spec.When {
				terms[i] = term.String()
			}

			name := mcp.ToolName(c)
			annotated[name] = pinnedTool{
				Access:     string(spec.Access),
				When:       strings.Join(terms, ","),
				Flags:      strings.Join(spec.Flags, ","),
				Properties: pinnedTools[name].Properties,
			}
		}

		for _, child := range c.Commands() {
			walk(child)
		}
	}
	walk(cmd.RootCmd())

	assert.DeepEqual(t, annotated, pinnedTools)

	ws := installWorkspace(t)
	reference := ws.withReference(t, mcpEnvFixture)

	t.Run("EverythingOn", func(t *testing.T) {
		session := ws.connect(t, "", reference, "WS_AI_MCP_ALLOW_WRITE=true", "WS_FEATURES_STORE_URL=http://127.0.0.1:1")

		tools := listTools(t, session)
		assert.DeepEqual(t, slices.Sorted(maps.Keys(tools)), slices.Sorted(maps.Keys(pinnedTools)))

		for name, tool := range tools {
			pinned := pinnedTools[name]

			assert.Equal(t, strings.Join(properties(t, tool), ","), pinned.Properties, name)
			assert.Equal(t, tool.Annotations.ReadOnlyHint, pinned.Access == "read", name)
			assert.Equal(t, tool.Annotations.DestructiveHint != nil && *tool.Annotations.DestructiveHint, pinned.Access == "write", name)
			assert.Assert(t, tool.Description != "" && tool.Title != "", name)
		}
	})

	t.Run("Default", func(t *testing.T) {
		tools := listTools(t, ws.connect(t, "", reference))

		var want []string
		for name, pinned := range pinnedTools {
			if pinned.Access == "read" && pinned.When == "" {
				want = append(want, name)
			}
		}

		slices.Sort(want)
		assert.DeepEqual(t, slices.Sorted(maps.Keys(tools)), want)
	})

	t.Run("Filtered", func(t *testing.T) {
		tools := listTools(t, ws.connect(t, "", reference, "WS_AI_MCP_ALLOW_WRITE=true", "WS_AI_MCP_TOOLS= info , editor_open "))

		assert.DeepEqual(t, slices.Sorted(maps.Keys(tools)), []string{"editor_open", "info_extensions", "info_metrics", "info_uptime", "info_version"})
	})

	t.Run("Typo", func(t *testing.T) {
		tools := listTools(t, ws.connect(t, "", reference, "WS_AI_MCP_ALLOW_WRITE=true", "WS_AI_MCP_TOOLS=bogus"))

		assert.Equal(t, len(tools), 0)
	})

	t.Run("UnparsableWriteToggle", func(t *testing.T) {
		tools := listTools(t, ws.connect(t, "", reference, "WS_AI_MCP_ALLOW_WRITE=maybe"))

		assert.Assert(t, tools["show_env"] != nil)
		assert.Assert(t, tools["editor_open"] == nil)
	})
}

func TestMcpToolsNeverLeakSecrets(t *testing.T) {
	ws := installWorkspace(t)

	canaries := map[string]string{
		"auth.password": "canary-from-the-environment",
		"auth.token":    "canary-from-a-file-reference",
		"auth.key":      "canary-from-the-convention-path",
	}

	tokenFile := filepath.Join(ws.dir, "token")
	assert.NilError(t, os.WriteFile(tokenFile, []byte(canaries["auth.token"]+"\n"), 0o600))

	secrets := filepath.Join(ws.dir, "secrets")
	assert.NilError(t, os.MkdirAll(filepath.Join(secrets, "auth"), 0o700))
	assert.NilError(t, os.WriteFile(filepath.Join(secrets, "auth", "key"), []byte(canaries["auth.key"]+"\n"), 0o600))

	session := ws.connect(t, "",
		ws.withReference(t, mcpEnvFixture),
		"WS_AI_MCP_ALLOW_WRITE=true",
		"WS_FEATURES_STORE_URL=http://127.0.0.1:1",
		"WS_AUTH_PASSWORD="+canaries["auth.password"],
		"WS_AUTH_TOKEN=file:"+tokenFile,
		"WS__INTERNAL_SECRETS_ROOT="+secrets,
		"CLICOLOR_FORCE=1",
		"WS__INTERNAL_FORCE_COLOR=1",
	)

	assertClean := func(t *testing.T, text string) {
		t.Helper()

		for _, canary := range canaries {
			assert.Assert(t, !strings.Contains(text, canary), "secret value in %q", text)
		}

		assert.Assert(t, !strings.Contains(text, "\x1b"), "escape in %q", text)
	}

	tools := listTools(t, session)
	assert.Equal(t, len(tools), len(pinnedTools))

	for name, tool := range tools {
		t.Run(name, func(t *testing.T) {
			arguments := map[string]any{}
			if slices.Contains(properties(t, tool), "args") {
				arguments["args"] = []string{"x"}
			}

			text, _ := callTool(t, session, name, arguments)
			assertClean(t, text)
		})
	}

	for key := range canaries {
		t.Run(key, func(t *testing.T) {
			text, failed := callTool(t, session, "show_env", map[string]any{"args": []string{key}})

			assertClean(t, text)
			assert.Assert(t, !failed, text)
			assert.Assert(t, strings.Contains(text, key), text)
			assert.Assert(t, strings.Contains(text, "<redacted>"), text)
		})
	}

	t.Run("UnreadableSecretFile", func(t *testing.T) {
		broken := ws.connect(t, "", ws.withReference(t, mcpEnvFixture), "WS_AUTH_TOKEN=file:"+filepath.Join(ws.dir, "missing"), "WS_AUTH_PASSWORD="+canaries["auth.password"])

		text, failed := callTool(t, broken, "show_env", map[string]any{"args": []string{"auth.token"}})

		assertClean(t, text)
		assert.Assert(t, failed, text)
		assert.Assert(t, strings.Contains(text, "missing"), text)
	})

	t.Run("ValueModeIsNotReachable", func(t *testing.T) {
		for _, arguments := range []map[string]any{
			{"args": []string{"auth.password"}, "value": true},
			{"args": []string{"--value", "auth.password"}},
			{"args": []string{"auth.password", "--value"}},
		} {
			text, failed := callTool(t, session, "show_env", arguments)

			assertClean(t, text)
			assert.Assert(t, failed, text)
		}
	})
}

type frame struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

type stdioServer struct {
	command *exec.Cmd
	stdin   io.WriteCloser
	lines   *bufio.Scanner
	stderr  bytes.Buffer
	frames  map[int]frame
}

func (w *workspace) start(t *testing.T, extraEnv ...string) *stdioServer {
	t.Helper()

	server := &stdioServer{command: w.server(extraEnv...), frames: map[int]frame{}}
	server.command.Stderr = &server.stderr

	stdin, err := server.command.StdinPipe()
	assert.NilError(t, err)
	stdout, err := server.command.StdoutPipe()
	assert.NilError(t, err)
	assert.NilError(t, server.command.Start())

	watchdog := time.AfterFunc(time.Minute, func() { _ = server.command.Process.Kill() })
	t.Cleanup(func() { watchdog.Stop() })

	server.stdin = stdin
	server.lines = bufio.NewScanner(stdout)
	server.lines.Buffer(nil, 1<<20)

	return server
}

func (s *stdioServer) send(t *testing.T, requests ...string) {
	t.Helper()

	_, err := fmt.Fprintln(s.stdin, strings.Join(requests, "\n"))
	assert.NilError(t, err)
}

func (s *stdioServer) read(t *testing.T, until int) {
	t.Helper()

	for (until == 0 || len(s.frames) < until) && s.lines.Scan() {
		var decoded frame
		assert.NilError(t, json.Unmarshal(s.lines.Bytes(), &decoded), "not a frame: %q", s.lines.Text())
		assert.Equal(t, decoded.JSONRPC, "2.0", s.lines.Text())
		assert.Assert(t, decoded.ID != nil, s.lines.Text())

		s.frames[*decoded.ID] = decoded
	}

	assert.NilError(t, s.lines.Err())
}

func (s *stdioServer) wait(t *testing.T) error {
	t.Helper()
	s.read(t, 0)

	return s.command.Wait()
}

const (
	initializeFrame  = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`
	initializedFrame = `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	listFrame        = `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
)

func TestServeMcpOverStdio(t *testing.T) {
	ws := installWorkspace(t)
	reference := ws.withReference(t, mcpEnvFixture)

	t.Run("EveryStdoutLineIsAFrame", func(t *testing.T) {
		server := ws.start(t, "WS_AI_MCP_ALLOW_WRITE=true", "CLICOLOR_FORCE=1")

		server.send(t, initializeFrame, initializedFrame, listFrame,
			`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"show_env","arguments":{"args":["server.port"]}}}`,
			`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"template_show","arguments":{"args":["nope"]}}}`,
			`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"editor_open","arguments":{"args":["main.go"]}}}`,
			`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"logs","arguments":{"follow":true}}}`,
		)
		server.read(t, 6)
		assert.NilError(t, server.stdin.Close())
		assert.NilError(t, server.wait(t), server.stderr.String())

		frames := server.frames
		assert.Equal(t, len(frames), 6, server.stderr.String())

		var initialized struct {
			ProtocolVersion string `json:"protocolVersion"`
			ServerInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
		}
		assert.NilError(t, json.Unmarshal(frames[1].Result, &initialized))
		assert.Equal(t, initialized.ProtocolVersion, "2025-11-25")
		assert.Equal(t, initialized.ServerInfo.Name, "ws-cli")
		assert.Equal(t, initialized.ServerInfo.Version, cmd.RootCmd().Version)
		assert.Assert(t, initialized.ServerInfo.Version != "")

		list := string(frames[2].Result)
		assert.Assert(t, strings.Contains(list, `"name":"show_env"`), list)
		assert.Assert(t, !strings.Contains(list, `"name":"editor_open"`), list)
		assert.Assert(t, !strings.Contains(list, `"name":"feature_store"`), list)

		value := string(frames[3].Result)
		assert.Assert(t, strings.Contains(value, "8080") && !strings.Contains(value, "isError"), value)

		missing := string(frames[4].Result)
		assert.Assert(t, strings.Contains(missing, `"isError":true`) && strings.Contains(missing, "Template 'nope' not found"), missing)

		assert.Assert(t, strings.Contains(string(frames[5].Error), "unknown tool"), string(frames[5].Error))

		injected := string(frames[6].Result)
		assert.Assert(t, strings.Contains(injected, `"isError":true`) && strings.Contains(injected, "unknown argument [follow]"), injected)

		for id, decoded := range frames {
			assert.Assert(t, !strings.Contains(string(decoded.Result), `\u001b`), "escape in frame %d", id)
		}
	})

	t.Run("WarnsOnStderrOnly", func(t *testing.T) {
		cases := map[string]struct {
			env  []string
			want []string
		}{
			"UnparsableToggle": {
				[]string{reference, "WS_AI_MCP_ALLOW_WRITE=maybe"},
				[]string{"Ignoring [WS_AI_MCP_ALLOW_WRITE], write tools stay off:"},
			},
			"UndeclaredVariables": {
				[]string{"WS_AI_MCP_ALLOW_WRITE=true", "WS_AI_MCP_TOOLS=info"},
				[]string{"Ignoring [WS_AI_MCP_ALLOW_WRITE]: it is not declared in this workspace", "Ignoring [WS_AI_MCP_TOOLS]: it is not declared in this workspace"},
			},
			"UnmatchedEntry": {
				[]string{reference, "WS_AI_MCP_TOOLS=bogus,info,editor_open"},
				[]string{"Ignoring [WS_AI_MCP_TOOLS] entry [bogus]: it names no tool or group"},
			},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				server := ws.start(t, tc.env...)

				server.send(t, initializeFrame, initializedFrame, listFrame)
				server.read(t, 2)
				assert.NilError(t, server.stdin.Close())
				assert.NilError(t, server.wait(t), server.stderr.String())

				for _, want := range tc.want {
					assert.Assert(t, strings.Contains(server.stderr.String(), want), server.stderr.String())
				}

				assert.Equal(t, strings.Count(server.stderr.String(), "Ignoring"), len(tc.want), server.stderr.String())
				assert.Assert(t, !strings.Contains(string(server.frames[2].Result), "Ignoring"))
				assert.Assert(t, strings.Contains(string(server.frames[2].Result), `"name":"info_version"`))
			})
		}
	})

	t.Run("NoWarningsByDefault", func(t *testing.T) {
		server := ws.start(t, reference)

		assert.NilError(t, server.stdin.Close())
		assert.NilError(t, server.wait(t))
		assert.Equal(t, server.stderr.String(), "")
	})

	for name, sent := range map[string]syscall.Signal{"SIGTERM": syscall.SIGTERM, "SIGINT": syscall.SIGINT, "SIGHUP": syscall.SIGHUP} {
		t.Run("StopsOn"+name, func(t *testing.T) {
			server := ws.start(t)

			server.send(t, initializeFrame)
			server.read(t, 1)
			assert.NilError(t, server.command.Process.Signal(sent))

			assert.NilError(t, server.wait(t), server.stderr.String())
			assert.Equal(t, server.stderr.String(), "")
		})
	}

	t.Run("SignalCancelsARunningCall", func(t *testing.T) {
		store, err := net.Listen("tcp", "127.0.0.1:0")
		assert.NilError(t, err)
		defer store.Close()

		reached := make(chan net.Conn, 8)
		go func() {
			for {
				conn, err := store.Accept()
				if err != nil {
					return
				}

				reached <- conn
			}
		}()

		server := ws.start(t, reference, "WS_FEATURES_STORE_URL=http://"+store.Addr().String())
		server.send(t, initializeFrame, initializedFrame, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"feature_store","arguments":{}}}`)
		server.read(t, 1)

		select {
		case conn := <-reached:
			defer conn.Close()
		case <-time.After(30 * time.Second):
			t.Fatal("the feature_store call never reached the store")
		}

		start := time.Now()
		assert.NilError(t, server.command.Process.Signal(syscall.SIGTERM))
		assert.NilError(t, server.wait(t), server.stderr.String())
		assert.Assert(t, time.Since(start) < 3*time.Second, "took %s", time.Since(start))
	})

	t.Run("ExitsCleanlyOnImmediateEOF", func(t *testing.T) {
		out, err := ws.server().Output()

		assert.NilError(t, err)
		assert.Equal(t, string(out), "")
	})

	t.Run("RefusesArgsAndFlags", func(t *testing.T) {
		out := ws.run(t, nil, "serve", "mcp", "extra")

		assert.Assert(t, strings.Contains(out, `Unknown command "extra"`), out)

		for _, flag := range []string{"--port", "--bind"} {
			out := ws.run(t, nil, "serve", "mcp", flag, "1")

			assert.Assert(t, strings.Contains(strings.ToLower(out), "unknown flag: "+flag), out)
		}
	})

	for _, protocol := range []string{"2025-11-25", "2026-07-28"} {
		t.Run("Protocol"+protocol, func(t *testing.T) {
			session := ws.connect(t, protocol)

			assert.Equal(t, session.InitializeResult().ProtocolVersion, protocol)
			assert.Assert(t, listTools(t, session)["show_path_home"] != nil)

			text, failed := callTool(t, session, "show_path_home", map[string]any{})
			assert.Assert(t, strings.Contains(text, "/workspace"), text)
			assert.Assert(t, !failed)
		})
	}
}
