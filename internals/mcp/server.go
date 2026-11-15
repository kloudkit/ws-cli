package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/kloudkit/ws-cli/internals/server"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "ws-cli"

var outcomes = map[bool]string{false: "ok", true: "failed"}

type Runner func(ctx context.Context, args []string) (text string, failed bool)

func NewServer(version string, tools []Tool, run Runner, log io.Writer) *sdk.Server {
	server := sdk.NewServer(
		&sdk.Implementation{Name: serverName, Version: version},
		&sdk.ServerOptions{Capabilities: &sdk.ServerCapabilities{Tools: &sdk.ToolCapabilities{}}},
	)

	for _, tool := range tools {
		server.AddTool(describe(tool), handle(tool, run, log))
	}

	return server
}

func Serve(ctx context.Context, version string, tools []Tool, log io.Writer) error {
	run := func(call context.Context, args []string) (string, bool) {
		call, cancel := context.WithCancel(call)
		defer cancel()
		defer context.AfterFunc(ctx, cancel)()

		return Run(call, args)
	}

	err := NewServer(version, tools, run, log).Run(ctx, &sdk.StdioTransport{})
	if errors.Is(err, context.Canceled) {
		return nil
	}

	return err
}

func describe(tool Tool) *sdk.Tool {
	annotations := &sdk.ToolAnnotations{ReadOnlyHint: tool.Access == Read}
	if tool.Access == Write {
		destructive := true
		annotations.DestructiveHint = &destructive
	}

	return &sdk.Tool{
		Name:        tool.Name,
		Title:       tool.Title,
		Description: tool.Description,
		InputSchema: tool.Schema(),
		Annotations: annotations,
	}
}

func handle(tool Tool, run Runner, log io.Writer) sdk.ToolHandler {
	return func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		start := time.Now()

		argv, err := tool.Argv(req.Params.Arguments)
		text, failed := fmt.Sprintf("invalid arguments for %s: %v", tool.Name, err), true
		if err == nil {
			text, failed = run(ctx, argv)
		}

		server.LogRequest(log, tool.Name, outcomes[failed], time.Since(start))

		return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: text}}, IsError: failed}, nil
	}
}
