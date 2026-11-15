package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "ws-cli"

type Runner func(ctx context.Context, args []string) (text string, failed bool)

func NewServer(version string, tools []Tool, run Runner) *sdk.Server {
	server := sdk.NewServer(
		&sdk.Implementation{Name: serverName, Version: version},
		&sdk.ServerOptions{Capabilities: &sdk.ServerCapabilities{Tools: &sdk.ToolCapabilities{}}},
	)

	for _, tool := range tools {
		server.AddTool(describe(tool), handle(tool, run))
	}

	return server
}

func Serve(ctx context.Context, version string, tools []Tool) error {
	run := func(call context.Context, args []string) (string, bool) {
		call, cancel := context.WithCancel(call)
		defer cancel()
		defer context.AfterFunc(ctx, cancel)()

		return Run(call, args)
	}

	err := NewServer(version, tools, run).Run(ctx, &sdk.StdioTransport{})
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

func handle(tool Tool, run Runner) sdk.ToolHandler {
	return func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		argv, err := tool.Argv(req.Params.Arguments)
		if err != nil {
			return result(fmt.Sprintf("invalid arguments for %s: %v", tool.Name, err), true), nil
		}

		return result(run(ctx, argv)), nil
	}
}

func result(text string, failed bool) *sdk.CallToolResult {
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: text}}, IsError: failed}
}
