package template

import (
	"github.com/spf13/cobra"
)

var TemplateCmd = &cobra.Command{
	Use:         "template",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Manage static configuration files",
	Long:        "Copy shared configuration files (linters, formatters) from their global locations into a project, and inspect what they hold.",
	Example: `# List available templates
ws template list

# Apply the ruff config to the current project
ws template apply ruff`,
}
