package mcp

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

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
				visible, err := When(spec.When)
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

func When(terms []Term) (bool, error) {
	for _, term := range terms {
		key := config.RuntimeKey(term.Group, term.Prop)

		prop, declared, err := config.LookupProperty(key)
		if err != nil {
			return false, err
		}

		if !declared {
			return false, nil
		}

		value, err := config.ResolveKey(key)
		if err != nil {
			return false, nil
		}

		truth := value != ""
		if truth && prop.Type == "boolean" {
			if truth, err = config.ParseBool(value); err != nil {
				return false, nil
			}
		}

		if truth == term.Negated {
			return false, nil
		}
	}

	return true, nil
}

func newTool(cmd *cobra.Command, spec Spec) (Tool, error) {
	tool := Tool{
		Name:        ToolName(cmd),
		Title:       cmd.Short,
		Description: cmp.Or(spec.Description, cmd.Long, cmd.Short),
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
	given := map[string]json.RawMessage{}
	if len(arguments) > 0 && string(arguments) != "null" {
		if err := json.Unmarshal(arguments, &given); err != nil {
			return nil, fmt.Errorf("arguments must be a JSON object")
		}
	}

	argv := slices.Clone(t.Command)

	for _, param := range t.Params {
		raw, ok := given[param.Name]
		if !ok {
			continue
		}
		delete(given, param.Name)

		value, err := param.parse(raw)
		if err != nil {
			return nil, err
		}

		argv = append(argv, "--"+param.Name+"="+value)
	}

	var positionals []string
	if raw, ok := given[argsParam]; ok && t.Args != "" {
		delete(given, argsParam)

		var err error
		if positionals, err = parseStrings(raw); err != nil {
			return nil, err
		}
	}

	if len(given) > 0 {
		unknown := slices.Sorted(maps.Keys(given))
		return nil, fmt.Errorf("unknown argument [%s] (accepted: %s)", unknown[0], strings.Join(t.accepted(), ", "))
	}

	if len(positionals) > 0 {
		argv = append(append(argv, "--"), positionals...)
	}

	return argv, nil
}

func (t Tool) accepted() []string {
	names := make([]string, len(t.Params))
	for i, param := range t.Params {
		names[i] = param.Name
	}

	if t.Args != "" {
		names = append(names, argsParam)
	}

	if len(names) == 0 {
		return []string{"none"}
	}

	return names
}

func (p Param) parse(raw json.RawMessage) (string, error) {
	text := string(raw)

	switch p.Type {
	case "boolean":
		if text == "true" || text == "false" {
			return text, nil
		}
	case "integer":
		if _, err := strconv.Atoi(text); err == nil {
			return text, nil
		}
	case "string":
		var value string
		if err := json.Unmarshal(raw, &value); err == nil && text != "null" && !hasNul(value) {
			return value, nil
		}
	}

	return "", fmt.Errorf("argument [%s] must be of type %s", p.Name, p.Type)
}

func parseStrings(raw json.RawMessage) ([]string, error) {
	var items []*string
	if err := json.Unmarshal(raw, &items); err != nil || string(raw) == "null" {
		return nil, fmt.Errorf("argument [%s] must be an array of strings", argsParam)
	}

	values := make([]string, len(items))
	for i, item := range items {
		if item == nil || hasNul(*item) {
			return nil, fmt.Errorf("argument [%s] must be an array of strings", argsParam)
		}

		values[i] = *item
	}

	return values, nil
}

func hasNul(value string) bool {
	return strings.ContainsRune(value, 0)
}
