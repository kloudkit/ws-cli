package docs

import (
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type Option struct {
	Name      string `yaml:"name"`
	Shorthand string `yaml:"shorthand,omitempty"`
	Default   string `yaml:"default,omitempty"`
	Usage     string `yaml:"usage,omitempty"`
}

type Command struct {
	Name        string    `yaml:"name"`
	Synopsis    string    `yaml:"synopsis,omitempty"`
	Description string    `yaml:"description,omitempty"`
	Usage       string    `yaml:"usage,omitempty"`
	Aliases     []string  `yaml:"aliases,omitempty"`
	Example     string    `yaml:"example,omitempty"`
	Options     []Option  `yaml:"options,omitempty"`
	Commands    []Command `yaml:"commands,omitempty"`
}

func Serialize(root *cobra.Command) ([]byte, error) {
	return yaml.Marshal(walk(root))
}

func walk(c *cobra.Command) Command {
	command := Command{
		Name:        c.CommandPath(),
		Synopsis:    c.Short,
		Description: c.Long,
		Aliases:     c.Aliases,
		Example:     c.Example,
	}

	if c.Runnable() {
		command.Usage = c.UseLine()
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

	children := append([]*cobra.Command(nil), c.Commands()...)
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name() < children[j].Name()
	})

	for _, child := range children {
		if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
			continue
		}

		command.Commands = append(command.Commands, walk(child))
	}

	return command
}
