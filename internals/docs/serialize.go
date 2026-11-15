package docs

import (
	"fmt"
	"path/filepath"

	"github.com/kloudkit/ws-cli/internals/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.yaml.in/yaml/v3"
)

type Option struct {
	Name      string `yaml:"name"`
	Shorthand string `yaml:"shorthand,omitempty"`
	Default   string `yaml:"default,omitempty"`
	Usage     string `yaml:"usage,omitempty"`
}

type MCP struct {
	Name        string   `yaml:"name"`
	Access      string   `yaml:"access"`
	When        []string `yaml:"when,omitempty"`
	Flags       []string `yaml:"flags,omitempty"`
	Description string   `yaml:"description,omitempty"`
}

type Command struct {
	Name        string    `yaml:"name"`
	Since       string    `yaml:"since,omitempty"`
	Deprecated  string    `yaml:"deprecated,omitempty"`
	Synopsis    string    `yaml:"synopsis,omitempty"`
	Description string    `yaml:"description,omitempty"`
	Usage       string    `yaml:"usage,omitempty"`
	Aliases     []string  `yaml:"aliases,omitempty"`
	Example     string    `yaml:"example,omitempty"`
	MCP         *MCP      `yaml:"mcp,omitempty"`
	Options     []Option  `yaml:"options,omitempty"`
	Commands    []Command `yaml:"commands,omitempty"`
}

func Serialize(root *cobra.Command) ([]byte, error) {
	command, err := walk(root)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(command)
}

func walk(c *cobra.Command) (Command, error) {
	command := Command{
		Name:        c.CommandPath(),
		Since:       c.Annotations["since"],
		Deprecated:  c.Annotations["deprecated"],
		Synopsis:    c.Short,
		Description: c.Long,
		Aliases:     c.Aliases,
		Example:     c.Example,
	}

	if command.Deprecated != "" {
		command.Since = ""
	}

	spec, annotated, err := mcp.ParseSpec(c.Annotations)
	if err != nil {
		return Command{}, fmt.Errorf("%s: %w", c.CommandPath(), err)
	}

	if c.Runnable() {
		command.Usage = c.UseLine()
	}

	if annotated && c.Runnable() {
		command.MCP = &MCP{Name: mcp.ToolName(c), Access: string(spec.Access), Flags: spec.Flags, Description: spec.Description}
		for _, term := range spec.When {
			command.MCP.When = append(command.MCP.When, term.String())
		}
	}

	c.NonInheritedFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}

		def := f.DefValue
		if filepath.IsAbs(def) {
			def = ""
		}

		command.Options = append(command.Options, Option{
			Name:      f.Name,
			Shorthand: f.Shorthand,
			Default:   def,
			Usage:     f.Usage,
		})
	})

	for _, child := range c.Commands() {
		if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
			continue
		}

		nested, err := walk(child)
		if err != nil {
			return Command{}, err
		}

		command.Commands = append(command.Commands, nested)
	}

	return command, nil
}
