package mcp

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/spf13/cobra"
)

const (
	argsParam = "args"
	allTools  = "*"
)

var toolsKey = config.RuntimeKey("ai", "mcp_tools")

var paramTypes = map[string]string{"bool": "boolean", "int": "integer", "string": "string"}

type Param struct {
	Name        string
	Type        string
	Description string
}

type Tool struct {
	Name        string
	Title       string
	Description string
	Access      Access
	Command     []string
	Params      []Param
	Args        string
}

type Settings struct {
	AllowWrite bool
	// A nil Tools lists every tool; an empty one lists none.
	Tools []string
}

func ResolveSettings() (Settings, []string) {
	var (
		settings Settings
		warnings []string
	)

	allowKey := config.RuntimeKey("ai", "mcp_allow_write")
	if declared(allowKey, &warnings) {
		value, err := config.ResolveKey(allowKey)
		if err == nil && value != "" {
			settings.AllowWrite, err = config.ParseBool(value)
		}

		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Ignoring [%s], write tools stay off: %v", allowKey, err))
		}
	}

	if declared(toolsKey, &warnings) {
		tools, err := config.ResolveListKey(toolsKey, ",")
		switch {
		case err != nil:
			settings.Tools = []string{}
			warnings = append(warnings, fmt.Sprintf("Cannot read [%s], serving no tools: %v", toolsKey, err))
		case !slices.Contains(tools, allTools):
			settings.Tools = tools
		}
	}

	return settings, warnings
}

func declared(key string, warnings *[]string) bool {
	_, found, err := config.LookupProperty(key)
	if err == nil && !found && env.String(key) != "" {
		*warnings = append(*warnings, fmt.Sprintf("Ignoring [%s]: it is not declared in this workspace", key))
	}

	return err == nil && found
}

func (s Settings) allows(tool Tool) bool {
	if tool.Access == Write && !s.AllowWrite {
		return false
	}

	return s.Tools == nil || slices.ContainsFunc(s.Tools, tool.matches)
}

func (t Tool) matches(entry string) bool {
	for depth := range t.Command {
		if entry == toolName(t.Command[:depth+1]) {
			return true
		}
	}

	return false
}

func BuildTools(root *cobra.Command, settings Settings) ([]Tool, []string, error) {
	var (
		tools    []Tool
		known    []Tool
		warnings []string
		seen     = map[string]string{}
	)

	var walk func(cmd *cobra.Command) error
	walk = func(cmd *cobra.Command) error {
		if cmd.Hidden {
			return nil
		}

		spec, annotated, err := ParseSpec(cmd.Annotations)
		if err != nil {
			return fmt.Errorf("%s: %w", cmd.CommandPath(), err)
		}

		if annotated && cmd.Runnable() {
			tool, err := newTool(cmd, spec)
			if err != nil {
				return fmt.Errorf("%s: %w", cmd.CommandPath(), err)
			}

			if other, taken := seen[tool.Name]; taken {
				return fmt.Errorf("%s: tool name [%s] is already taken by %s", cmd.CommandPath(), tool.Name, other)
			}
			seen[tool.Name] = cmd.CommandPath()
			known = append(known, tool)

			if settings.allows(tool) {
				visible, err := config.Holds(spec.When)
				if err != nil {
					return fmt.Errorf("%s: %w", cmd.CommandPath(), err)
				}

				if visible {
					tools = append(tools, tool)
				}
			}
		}

		for _, child := range cmd.Commands() {
			if err := walk(child); err != nil {
				return err
			}
		}

		return nil
	}

	if err := walk(root); err != nil {
		return nil, nil, err
	}

	for _, entry := range slices.Compact(slices.Sorted(slices.Values(settings.Tools))) {
		if !slices.ContainsFunc(known, func(tool Tool) bool { return tool.matches(entry) }) {
			warnings = append(warnings, fmt.Sprintf("Ignoring [%s] entry [%s]: it names no tool or group", toolsKey, entry))
		}
	}

	return tools, warnings, nil
}

func newTool(cmd *cobra.Command, spec Spec) (Tool, error) {
	tool := Tool{
		Name:        ToolName(cmd),
		Title:       cmd.Short,
		Description: cmp.Or(cmd.Long, cmd.Short),
		Access:      spec.Access,
		Command:     commandPath(cmd),
		Args:        strings.TrimSpace(strings.TrimPrefix(cmd.Use, cmd.Name())),
	}

	for _, name := range spec.Flags {
		flag := cmd.Flag(name)
		if flag == nil {
			return Tool{}, fmt.Errorf("annotation [%s] names an unknown flag [%s]", AnnotationFlags, name)
		}

		kind, supported := paramTypes[flag.Value.Type()]
		if !supported || name == argsParam {
			return Tool{}, fmt.Errorf("annotation [%s] cannot expose flag [%s] of type %s", AnnotationFlags, name, flag.Value.Type())
		}

		if slices.ContainsFunc(tool.Params, func(p Param) bool { return p.Name == name }) {
			return Tool{}, fmt.Errorf("annotation [%s] names flag [%s] twice", AnnotationFlags, name)
		}

		tool.Params = append(tool.Params, Param{Name: name, Type: kind, Description: flag.Usage})
	}

	return tool, nil
}

func (t Tool) Schema() json.RawMessage {
	properties := map[string]any{}
	for _, param := range t.Params {
		properties[param.Name] = map[string]any{"type": param.Type, "description": param.Description}
	}

	if t.Args != "" {
		properties[argsParam] = map[string]any{
			"type":        "array",
			"items":       map[string]any{"type": "string"},
			"description": "Positional arguments: " + t.Args,
		}
	}

	schema, _ := json.Marshal(map[string]any{"type": "object", "properties": properties, "additionalProperties": false})

	return schema
}

func (t Tool) Argv(arguments json.RawMessage) ([]string, error) {
	var schema jsonschema.Schema
	if err := json.Unmarshal(t.Schema(), &schema); err != nil {
		return nil, err
	}

	resolved, err := schema.Resolve(nil)
	if err != nil {
		return nil, err
	}

	given := map[string]any{}
	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &given); err != nil {
			return nil, fmt.Errorf("arguments must be a JSON object")
		}
	}

	if err := resolved.Validate(given); err != nil {
		return nil, err
	}

	argv := slices.Clone(t.Command)

	for _, param := range t.Params {
		switch value := given[param.Name].(type) {
		case nil:
		case float64:
			argv = append(argv, "--"+param.Name+"="+strconv.FormatFloat(value, 'f', -1, 64))
		default:
			argv = append(argv, fmt.Sprintf("--%s=%v", param.Name, value))
		}
	}

	if positionals, _ := given[argsParam].([]any); len(positionals) > 0 {
		argv = append(argv, "--")
		for _, positional := range positionals {
			argv = append(argv, positional.(string))
		}
	}

	if slices.ContainsFunc(argv, func(arg string) bool { return strings.ContainsRune(arg, 0) }) {
		return nil, fmt.Errorf("arguments must not hold a NUL byte")
	}

	return argv, nil
}
