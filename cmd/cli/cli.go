package cli

import (
	"strings"

	"github.com/kloudkit/ws-cli/internals/mcp"
	"github.com/spf13/cobra"
)

type Run func(cmd *cobra.Command, args []string) error

type MCPOption func(annotations map[string]string)

const (
	MCPRead  = mcp.Read
	MCPWrite = mcp.Write
)

func New(use, since, short, long string, run Run) *cobra.Command {
	return &cobra.Command{Use: use, Annotations: map[string]string{"since": since}, Short: short, Long: long, RunE: run}
}

func Group(use, since, short, long, example string, children ...*cobra.Command) *cobra.Command {
	cmd := New(use, since, short, long, nil)
	cmd.Example = example
	cmd.AddCommand(children...)

	return cmd
}

func MCP(cmd *cobra.Command, access mcp.Access, opts ...MCPOption) *cobra.Command {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}

	cmd.Annotations[mcp.AnnotationAccess] = string(access)

	for _, opt := range opts {
		opt(cmd.Annotations)
	}

	return cmd
}

func MCPWhen(terms ...string) MCPOption {
	return func(annotations map[string]string) { annotations[mcp.AnnotationWhen] = strings.Join(terms, ",") }
}

func MCPFlags(names ...string) MCPOption {
	return func(annotations map[string]string) { annotations[mcp.AnnotationFlags] = strings.Join(names, ",") }
}

func Bool(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}

func String(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func Int(cmd *cobra.Command, name string) int {
	v, _ := cmd.Flags().GetInt(name)
	return v
}
