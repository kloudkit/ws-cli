package template

import (
	"os"

	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/kloudkit/ws-cli/internals/template"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:         "list",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "List all available configuration templates",
	Long:        "List the templates you can apply, with their source paths and whether one is already applied here.",
	Aliases:     []string{"ls"},
	RunE:        runList,
}

func runList(cmd *cobra.Command, args []string) error {
	templates := template.SupportedTemplates
	names := template.GetTemplateNames()

	out := cmd.OutOrStdout()

	styles.PrintTitleCount(out, "Templates Available", len(names))

	items := make([]styles.DescriptionItem, len(names))
	for i, name := range names {
		config := templates[name]
		description := path.ShortenHomePath(path.ResolveConfigPath(config.SourcePath))

		if _, err := os.Stat(config.OutputName); err == nil {
			description += " " + styles.Success().Render("(applied)")
		}

		items[i] = styles.DescriptionItem{Name: name, Description: description}
	}

	styles.PrintList(out, items)

	styles.PrintHints(out,
		styles.Hint{Command: "ws template apply <name>", Description: "Apply a template"},
		styles.Hint{Command: "ws template show <name>", Description: "View template contents"},
		styles.Hint{Command: "ws template show --local <name>", Description: "View applied template"},
	)

	return nil
}

func init() {
	TemplateCmd.AddCommand(listCmd)
}
